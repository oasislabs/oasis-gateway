package event

import (
	"context"
	"net/url"

	auth "github.com/oasislabs/developer-gateway/auth/core"
	backend "github.com/oasislabs/developer-gateway/backend/core"
	"github.com/oasislabs/developer-gateway/errors"
	"github.com/oasislabs/developer-gateway/log"
	"github.com/oasislabs/developer-gateway/rpc"
)

type Services struct {
	Logger  log.Logger
	Request *backend.RequestManager
}

// EventHandler implements the handlers associated with subscriptions and
// event polling
type EventHandler struct {
	logger  log.Logger
	request *backend.RequestManager
}

// Subscribe creates a new subscription for the client on the required
// topics
func (h EventHandler) Subscribe(ctx context.Context, v interface{}) (interface{}, error) {
	authID := ctx.Value(auth.ContextKeyAuthID).(string)
	req := v.(*SubscribeRequest)

	if len(req.Events) == 0 {
		err := errors.New(errors.ErrEmptyInput, stderr.New("no events set on request"))
		h.logger.Debug(ctx, "failed to handle request", log.MapFields{
			"call_type": "SubscribeFailure",
		}, err)
		return 0, err
	}

	if len(req.Events) > 1 {
		err := errors.New(errors.ErrEmptyInput, stderr.New("only one event supported"))
		h.logger.Debug(ctx, "failed to handle request", log.MapFields{
			"call_type": "SubscribeFailure",
		}, err)
		return 0, err
	}

	query, err := url.ParseQuery(req.Filter)
	if err != nil {
		err := errors.New(errors.ErrParseQueryParams, err)
		h.logger.Debug(ctx, "failed to handle request", log.MapFields{
			"call_type": "SubscribeFailure",
		}, err)
		return 0, err
	}

	address := query.Get("address")
	if len(address) == 0 {
		err := errors.New(errors.ErrParseQueryParams, err)
		h.logger.Debug(ctx, "failed to handle request", log.MapFields{
			"call_type": "SubscribeFailure",
		}, err)
		return 0, err
	}

	id, err := h.request.Subscribe(ctx, backend.SubscribeRequest{
		Topic:   req.Events[0],
		Address: address,
	})

	return SubscribeResponse{
		ID: id,
	}, nil
}

// Unsubscribe destroys an existing client subscription and all the
// resources associated with it
func (h EventHandler) Unsubscribe(ctx context.Context, v interface{}) (interface{}, error) {
	_ = v.(*UnsubscribeRequest)
	return nil, errors.New(errors.ErrAPINotImplemented, nil)
}

// EventPoll allows the user to query for new events associated
// with a specific subscription
func (h EventHandler) EventPoll(ctx context.Context, v interface{}) (interface{}, error) {
	_ = v.(*EventPollRequest)
	return nil, errors.New(errors.ErrAPINotImplemented, nil)
}

// BindHandler binds the service handler to the provided
// HandlerBinder
func BindHandler(services Services, binder rpc.HandlerBinder) {
	if services.Request == nil {
		panic("Request must be provided as a service")
	}
	if services.Logger == nil {
		panic("Logger must be provided as a service")
	}

	handler := EventHandler{
		logger:  services.Logger.ForClass("event", "handler"),
		request: services.Request,
	}

	binder.Bind("POST", "/v0/api/event/subscribe", rpc.HandlerFunc(handler.Subscribe),
		rpc.EntityFactoryFunc(func() interface{} { return &SubscribeRequest{} }))
	binder.Bind("POST", "/v0/api/event/unsubscribe", rpc.HandlerFunc(handler.Unsubscribe),
		rpc.EntityFactoryFunc(func() interface{} { return &UnsubscribeRequest{} }))
	binder.Bind("POST", "/v0/api/event/poll", rpc.HandlerFunc(handler.EventPoll),
		rpc.EntityFactoryFunc(func() interface{} { return &EventPollRequest{} }))
}
