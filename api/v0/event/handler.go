package event

import (
	"context"

	"github.com/oasislabs/developer-gateway/errors"
	"github.com/oasislabs/developer-gateway/rpc"
)

// EventHandler implements the handlers associated with subscriptions and
// event polling
type EventHandler struct{}

// Subscribe creates a new subscription for the client on the required
// topics
func (h EventHandler) Subscribe(ctx context.Context, v interface{}) (interface{}, error) {
	_ = v.(*SubscribeRequest)
	return nil, rpc.HttpNotImplemented(ctx, errors.New(errors.ErrAPINotImplemented, nil))
}

// Unsubscribe destroys an existing client subscription and all the
// resources associated with it
func (h EventHandler) Unsubscribe(ctx context.Context, v interface{}) (interface{}, error) {
	_ = v.(*UnsubscribeRequest)
	return nil, rpc.HttpNotImplemented(ctx, errors.New(errors.ErrAPINotImplemented, nil))
}

// EventPoll allows the user to query for new events associated
// with a specific subscription
func (h EventHandler) EventPoll(ctx context.Context, v interface{}) (interface{}, error) {
	_ = v.(*EventPollRequest)
	return nil, rpc.HttpNotImplemented(ctx, errors.New(errors.ErrAPINotImplemented, nil))
}

// BindHandler binds the service handler to the provided
// HandlerBinder
func BindHandler(binder rpc.HandlerBinder) {
	handler := EventHandler{}

	binder.Bind("POST", "/v0/api/event/subscribe", rpc.HandlerFunc(handler.Subscribe),
		rpc.EntityFactoryFunc(func() interface{} { return &SubscribeRequest{} }))
	binder.Bind("POST", "/v0/api/event/unsubscribe", rpc.HandlerFunc(handler.Unsubscribe),
		rpc.EntityFactoryFunc(func() interface{} { return &UnsubscribeRequest{} }))
	binder.Bind("POST", "/v0/api/event/poll", rpc.HandlerFunc(handler.EventPoll),
		rpc.EntityFactoryFunc(func() interface{} { return &EventPollRequest{} }))
}
