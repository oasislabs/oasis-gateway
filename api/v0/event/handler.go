package event

import (
	"context"
	stderr "errors"
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
	authData := ctx.Value(auth.ContextAuthDataKey).(auth.AuthData)
	req := v.(*SubscribeRequest)

	if len(req.Events) == 0 {
		err := errors.New(errors.ErrEmptyInput, stderr.New("no events set on request"))
		h.logger.Debug(ctx, "failed to handle request", log.MapFields{
			"call_type": "SubscribeFailure",
		}, err)
		return nil, err
	}

	if len(req.Events) > 1 {
		err := errors.New(errors.ErrEmptyInput, stderr.New("only one event supported"))
		h.logger.Debug(ctx, "failed to handle request", log.MapFields{
			"call_type": "SubscribeFailure",
		}, err)
		return nil, err
	}

	query, derr := url.ParseQuery(req.Filter)
	if derr != nil {
		err := errors.New(errors.ErrParseQueryParams, derr)
		h.logger.Debug(ctx, "failed to handle request", log.MapFields{
			"call_type": "SubscribeFailure",
		}, err)
		return nil, err
	}

	address := query.Get("address")
	if len(address) == 0 {
		err := errors.New(errors.ErrParseQueryParams, nil)
		h.logger.Debug(ctx, "request does not contain address", log.MapFields{
			"call_type": "SubscribeFailure",
		}, err)
		return nil, err
	}

	id, err := h.request.Subscribe(ctx, backend.SubscribeRequest{
		Topic:      req.Events[0],
		Address:    address,
		SessionKey: authData.sessionKey,
	})
	if err != nil {
		h.logger.Debug(ctx, "failed to subscribe", log.MapFields{
			"call_type": "SubscribeFailure",
		}, err)
		return nil, err
	}

	return SubscribeResponse{
		ID: id,
	}, nil
}

// Unsubscribe destroys an existing client subscription and all the
// resources associated with it
func (h EventHandler) Unsubscribe(ctx context.Context, v interface{}) (interface{}, error) {
	authData := ctx.Value(auth.ContextAuthDataKey).(auth.AuthData)
	req := v.(*UnsubscribeRequest)

	err := h.request.Unsubscribe(ctx, backend.UnsubscribeRequest{
		ID:         req.ID,
		SessionKey: authData.sessionKey,
	})
	if err != nil {
		h.logger.Debug(ctx, "failed unsubscribe from events", log.MapFields{
			"call_type": "PollEventFailure",
			"id":        req.ID,
		}, err)
		return nil, err
	}

	return nil, nil
}

// EventPoll allows the user to query for new events associated
// with a specific subscription
func (h EventHandler) PollEvent(ctx context.Context, v interface{}) (interface{}, error) {
	authData := ctx.Value(auth.ContextAuthDataKey).(auth.AuthData)
	req := v.(*PollEventRequest)

	res, err := h.request.PollEvent(ctx, backend.PollEventRequest{
		DiscardPrevious: req.DiscardPrevious,
		Count:           req.Count,
		Offset:          req.Offset,
		ID:              req.ID,
		SessionKey:      authData.sessionKey,
	})
	if err != nil {
		h.logger.Debug(ctx, "failed to poll events from subscription", log.MapFields{
			"call_type": "PollEventFailure",
			"id":        req.ID,
		}, err)
		return nil, err
	}

	events := make([]Event, 0, len(res.Events))
	for _, r := range res.Events {
		switch r := r.(type) {
		case backend.ErrorEvent:
			events = append(events, ErrorEvent{
				ID:    r.ID,
				Cause: r.Cause,
			})
		case backend.DataEvent:
			events = append(events, DataEvent{
				ID:   r.ID,
				Data: r.Data,
			})
		default:
			panic("received unexpected event type from polling service")
		}
	}

	return PollEventResponse{
		Offset: res.Offset,
		Events: events,
	}, nil
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
	binder.Bind("POST", "/v0/api/event/poll", rpc.HandlerFunc(handler.PollEvent),
		rpc.EntityFactoryFunc(func() interface{} { return &PollEventRequest{} }))
}
