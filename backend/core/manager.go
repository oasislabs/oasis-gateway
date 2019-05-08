package core

import (
	"context"
	"errors"

	mqueue "github.com/oasislabs/developer-gateway/mqueue/core"
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
	GetPublicKeyService(context.Context, GetPublicKeyServiceRequest) (GetPublicKeyServiceResponse, error)
	ExecuteService(context.Context, uint64, ExecuteServiceRequest) Event
	DeployService(context.Context, uint64, DeployServiceRequest) Event
}

// RequestManager handles the client RPC requests. Most requests
// are asynchronous and they are handled by returning an identifier
// that the caller can later on query on to find out the outcome
// of the request.
type RequestManager struct {
	mqueue mqueue.MQueue
	client Client
}

type RequestManagerProperties struct {
	MQueue mqueue.MQueue
	Client Client
}

// NewRequestManager creates a new instance of a request manager
func NewRequestManager(properties RequestManagerProperties) *RequestManager {
	if properties.MQueue == nil {
		panic("MQueue must be set")
	}

	if properties.Client == nil {
		panic("Client must be set")
	}

	return &RequestManager{
		mqueue: properties.MQueue,
		client: properties.Client,
	}
}

// GetPublicKeyService retrieves the public key for a specific service
func (m *RequestManager) GetPublicKeyService(ctx context.Context, req GetPublicKeyServiceRequest) (GetPublicKeyServiceResponse, error) {
	if len(req.Address) == 0 {
		return GetPublicKeyServiceResponse{}, errors.New("address cannot be empty")
	}

	return m.client.GetPublicKeyService(ctx, req)
}

// RequestManager starts a request and provides an identifier for the caller to
// find the request later on. Executes an operation on a service
func (m *RequestManager) ExecuteServiceAsync(ctx context.Context, req ExecuteServiceRequest) (uint64, error) {
	if len(req.Address) == 0 {
		return 0, errors.New("address cannot be empty")
	}

	id, err := m.mqueue.Next(req.Key)
	if err != nil {
		return 0, err
	}

	go m.doRequest(ctx, req.Key, id, func() Event { return m.client.ExecuteService(ctx, id, req) })

	return id, nil
}

// RequestManager starts a request and provides an identifier for the caller to
// find the request later on. Deploys a new service
func (m *RequestManager) DeployServiceAsync(ctx context.Context, req DeployServiceRequest) (uint64, error) {
	id, err := m.mqueue.Next(req.Key)
	if err != nil {
		return 0, err
	}

	go m.doRequest(ctx, req.Key, id, func() Event { return m.client.DeployService(ctx, id, req) })

	return id, nil
}

func (m *RequestManager) doRequest(ctx context.Context, key string, id uint64, fn func() Event) {
	// TODO(stan): we should handle the case in which the request takes too long
	ev := fn()
	m.mqueue.Insert(key, mqueue.Element{
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
