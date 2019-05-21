package core

import (
	"context"
	"fmt"

	"github.com/oasislabs/developer-gateway/errors"
	"github.com/oasislabs/developer-gateway/log"
	mqueue "github.com/oasislabs/developer-gateway/mqueue/core"
	"github.com/oasislabs/developer-gateway/rpc"
)

type Event interface {
	EventID() uint64
}

type Events struct {
	Offset uint64
	Events []Event
}

// Client is an interface for any type that sends requests and
// receives responses
type Client interface {
	GetPublicKeyService(context.Context, GetPublicKeyServiceRequest) (*GetPublicKeyServiceResponse, errors.Err)
	ExecuteService(context.Context, uint64, ExecuteServiceRequest) (*ExecuteServiceResponse, errors.Err)
	DeployService(context.Context, uint64, DeployServiceRequest) (*DeployServiceResponse, errors.Err)
	SubscribeRequest(context.Context, CreateSubscriptionRequest, chan<- interface{}) errors.Err
	UnsubscribeRequest(context.Context, DestroySubscriptionRequest) errors.Err
}

// RequestManager handles the client RPC requests. Most requests
// are asynchronous and they are handled by returning an identifier
// that the caller can later on query on to find out the outcome
// of the request.
type RequestManager struct {
	mqueue mqueue.MQueue
	client Client
	logger log.Logger
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
		client: properties.Client,
		logger: properties.Logger,
	}
}

// GetPublicKeyService retrieves the public key for a specific service
func (m *RequestManager) GetPublicKeyService(
	ctx context.Context,
	req GetPublicKeyServiceRequest,
) (*GetPublicKeyServiceResponse, errors.Err) {
	if len(req.Address) == 0 {
		return nil, errors.New(errors.ErrInvalidAddress, nil)
	}

	return m.client.GetPublicKeyService(ctx, req)
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

	id, err := m.mqueue.Next(req.Key)
	if err != nil {
		return 0, err
	}

	go m.doRequest(ctx, req.Key, id, func() (Event, errors.Err) { return m.client.ExecuteService(ctx, id, req) })

	return id, nil
}

// RequestManager starts a request and provides an identifier for the caller to
// find the request later on. Deploys a new service
func (m *RequestManager) DeployServiceAsync(ctx context.Context, req DeployServiceRequest) (uint64, errors.Err) {
	id, err := m.mqueue.Next(req.Key)
	if err != nil {
		return 0, err
	}

	go m.doRequest(ctx, req.Key, id, func() (Event, errors.Err) { return m.client.DeployService(ctx, id, req) })

	return id, nil
}

// Subscribe creates a new subscription using the underlying backend and
// allocates the necessary resources from the store
func (m *RequestManager) Subscribe(ctx context.Context, req SubscribeRequest) (uint64, errors.Err) {
	// use a queue per queue to manage the number of queues created. This
	// also helps us with managing the resources a specific client is using
	key := req.Key + "-queue"
	id, err := m.mqueue.Next(key)
	if err != nil {
		return 0, err
	}

	if err := m.subscribe(ctx, id, req); err != nil {
		return 0, err
	}

	return id, nil
}

func (m *RequestManager) subscribe(ctx context.Context, id uint64, req SubscribeRequest) errors.Err {
	subID := fmt.Sprintf("%s-%d", req.Key, id)
	// TODO(stan): a request manager should have a context from which the subscription contexts
	// should derive
	sub := newSubscription(context.Background(), m.logger, m.mqueue, subID)
	if err := m.client.SubscribeRequest(ctx, CreateSubscriptionRequest{
		Topic:   req.Topic,
		Address: req.Address,
		SubID:   subID,
	}, sub.C); err != nil {
		return err
	}

	go sub.Start()
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

	// TODO(stan): in case of error, we should log the error. We should think if there's
	// a way to report the error in this case. A failure here means that a client will not
	// receive a response (not even a failure response)
	_ = m.mqueue.Insert(key, mqueue.Element{
		Value:  ev,
		Offset: id,
	})
}

// GetResponses retrieves the responses the RequestManager already got
// from the asynchronous requests.
func (m *RequestManager) GetResponses(key string, offset uint64, count uint) (Events, error) {
	els, err := m.mqueue.Retrieve(key, offset, count)
	if err != nil {
		return Events{}, err
	}

	var events []Event
	for _, el := range els.Elements {
		events = append(events, el.Value.(Event))
	}

	return Events{Offset: els.Offset, Events: events}, nil
}

// DiscardResponses discards responses stored by the RequestManager to make space
// for new requests
func (m *RequestManager) DiscardResponses(key string, offset uint64) error {
	return m.mqueue.Discard(key, offset)
}
