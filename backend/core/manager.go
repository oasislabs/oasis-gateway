package core

import (
	"context"
	stderr "errors"
	"fmt"

	"github.com/oasislabs/developer-gateway/errors"
	"github.com/oasislabs/developer-gateway/log"
	mqueue "github.com/oasislabs/developer-gateway/mqueue/core"
	"github.com/oasislabs/developer-gateway/rpc"
	"github.com/oasislabs/developer-gateway/stats"
)

// Client is an interface for any type that sends requests and
// receives responses
type Client interface {
	Name() string
	Stats() stats.Metrics
	GetPublicKey(context.Context, GetPublicKeyRequest) (GetPublicKeyResponse, errors.Err)
	ExecuteService(context.Context, uint64, ExecuteServiceRequest) (ExecuteServiceResponse, errors.Err)
	DeployService(context.Context, uint64, DeployServiceRequest) (DeployServiceResponse, errors.Err)
	SubscribeRequest(context.Context, CreateSubscriptionRequest, chan<- interface{}) errors.Err
	UnsubscribeRequest(context.Context, DestroySubscriptionRequest) errors.Err
}

// RequestManager handles the client RPC requests. Most requests
// are asynchronous and they are handled by returning an identifier
// that the caller can later on query to find out the outcome
// of the request.
type RequestManager struct {
	mqueue mqueue.MQueue
	client Client
	logger log.Logger
	subman *SubscriptionManager
}

func (r *RequestManager) Name() string {
	return "backend.core.RequestManager"
}

func (r *RequestManager) Stats() stats.Metrics {
	return stats.Metrics{
		"subscriptions": r.subman.Stats(),
	}
}

type RequestManagerProperties struct {
	MQueue mqueue.MQueue
	Client Client
	Logger log.Logger
}

// NewRequestManager creates a new instance of a request manager
func NewRequestManager(properties RequestManagerProperties) *RequestManager {
	if properties.MQueue == nil {
		panic("MQueue must be set")
	}

	if properties.Client == nil {
		panic("Client must be set")
	}

	if properties.Logger == nil {
		panic("Logger must be set")
	}

	return &RequestManager{
		mqueue: properties.MQueue,
		logger: properties.Logger,
		client: properties.Client,
		subman: NewSubscriptionManager(SubscriptionManagerProps{
			Context: context.Background(),
			Logger:  properties.Logger,
			MQueue:  properties.MQueue,
		}),
	}
}

// GetPublicKey retrieves the public key for a specific service
func (m *RequestManager) GetPublicKey(
	ctx context.Context,
	req GetPublicKeyRequest,
) (GetPublicKeyResponse, errors.Err) {
	if len(req.Address) == 0 {
		return GetPublicKeyResponse{}, errors.New(errors.ErrInvalidAddress, nil)
	}

	return m.client.GetPublicKey(ctx, req)
}

// RequestManager starts a request and provides an identifier for the caller to
// find the request later on. Executes an operation on a service
func (m *RequestManager) ExecuteServiceAsync(
	ctx context.Context,
	req ExecuteServiceRequest,
) (uint64, errors.Err) {
	if len(req.Address) == 0 {
		return 0, errors.New(errors.ErrInvalidAddress, nil)
	}

	id, err := m.mqueue.Next(ctx, mqueue.NextRequest{Key: req.SessionKey})
	if err != nil {
		return 0, errors.New(errors.ErrQueueNext, err)
	}

	go m.doRequest(ctx, req.SessionKey, id, func() (Event, errors.Err) { return m.client.ExecuteService(ctx, id, req) })

	return id, nil
}

// RequestManager starts a request and provides an identifier for the caller to
// find the request later on. Deploys a new service
func (m *RequestManager) DeployServiceAsync(ctx context.Context, req DeployServiceRequest) (uint64, errors.Err) {
	id, err := m.mqueue.Next(ctx, mqueue.NextRequest{Key: req.SessionKey})
	if err != nil {
		return 0, errors.New(errors.ErrQueueNext, err)
	}

	go m.doRequest(ctx, req.SessionKey, id, func() (Event, errors.Err) { return m.client.DeployService(ctx, id, req) })

	return id, nil
}

// Unsubscribe from an existing subscription freeing all the associated
// resources. After this operation all events from the subscription stream
// will be lost.
func (m *RequestManager) Unsubscribe(ctx context.Context, req UnsubscribeRequest) errors.Err {
	if len(req.SessionKey) == 0 {
		return errors.New(errors.ErrInvalidKey, stderr.New("key cannot be empty"))
	}

	subID := SubID(req.SessionKey, req.ID)
	if !m.subman.Exists(ctx, subID) {
		return errors.New(errors.ErrSubscriptionNotFound, stderr.New("cannot unsubscribe from subscription that does not exist"))
	}

	if err := m.client.UnsubscribeRequest(ctx, DestroySubscriptionRequest{
		SubID: subID,
	}); err != nil {
		return err
	}

	return m.subman.Destroy(ctx, subID)
}

// Subscribe creates a new subscription using the underlying backend and
// allocates the necessary resources from the store
func (m *RequestManager) Subscribe(ctx context.Context, req SubscribeRequest) (uint64, errors.Err) {
	if len(req.SessionKey) == 0 {
		return 0, errors.New(errors.ErrInvalidKey, stderr.New("key cannot be empty"))
	}

	// use a queue per subscription to manage the number of queues created. This
	// also helps us with managing the resources a specific client is using
	key := SubinfoID(req.SessionKey)
	id, err := m.mqueue.Next(ctx, mqueue.NextRequest{Key: key})
	if err != nil {
		return 0, errors.New(errors.ErrQueueNext, err)
	}

	if err := m.subscribe(ctx, id, req); err != nil {
		return 0, err
	}

	return id, nil
}

func (m *RequestManager) subscribe(ctx context.Context, id uint64, req SubscribeRequest) errors.Err {
	subID := SubID(req.SessionKey, id)
	// TODO(stan): a request manager should have a context from which the subscription contexts
	// should derive
	c := make(chan interface{}, 64)
	if err := m.subman.Create(ctx, subID, c); err != nil {
		return err
	}

	if err := m.client.SubscribeRequest(ctx, CreateSubscriptionRequest{
		Event:   req.Event,
		Address: req.Address,
		SubID:   subID,
		Topics:  req.Topics,
	}, c); err != nil {
		return err
	}

	return nil
}

func (m *RequestManager) doRequest(ctx context.Context, key string, id uint64, fn func() (Event, errors.Err)) {
	// TODO(stan): we should handle the case in which the request takes too long
	ev, err := fn()
	if err != nil {
		ev = ErrorEvent{
			ID: id,
			Cause: rpc.Error{
				ErrorCode:   err.ErrorCode().Code(),
				Description: err.ErrorCode().Desc(),
			},
		}
	}

	el, derr := makeElement(ev, id)
	if derr != nil {
		panic(fmt.Sprintf("failed to marshal event %s", derr.Error()))
	}

	if err := m.mqueue.Insert(ctx, mqueue.InsertRequest{Key: key, Element: el}); err != nil {
		panic(fmt.Sprintf("failed to insert event %s", err.Error()))
	}
}

// PollService retrieves the responses the RequestManager already got
// from the asynchronous requests.
func (m *RequestManager) PollService(ctx context.Context, req PollServiceRequest) (Events, errors.Err) {
	events, err := m.poll(ctx, req.SessionKey, req.Offset, req.Count, req.DiscardPrevious)
	return events, err
}

// PollEvent retrieves the responses the RequestManager already got
// from the asynchronous requests.
func (m *RequestManager) PollEvent(ctx context.Context, req PollEventRequest) (Events, errors.Err) {
	subID := SubID(req.SessionKey, req.ID)
	return m.poll(ctx, subID, req.Offset, req.Count, req.DiscardPrevious)
}

func (m *RequestManager) poll(ctx context.Context, key string, offset uint64, count uint, discardPrevious bool) (Events, errors.Err) {
	els, err := m.mqueue.Retrieve(ctx, mqueue.RetrieveRequest{Key: key, Offset: offset, Count: count})
	if err != nil {
		return Events{}, errors.New(errors.ErrQueueRetrieve, err)
	}

	if discardPrevious {
		if err := m.mqueue.Discard(ctx, mqueue.DiscardRequest{Key: key, Offset: offset}); err != nil {
			return Events{}, errors.New(errors.ErrQueueDiscard, err)
		}
	}

	var events []Event
	for _, el := range els.Elements {
		ev, err := deserializeElement(el)
		if err != nil {
			return Events{}, err
		}

		events = append(events, ev)
	}

	return Events{Offset: els.Offset, Events: events}, nil
}
