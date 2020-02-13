package event

import (
	"context"
	stderr "errors"
	"net/url"

	auth "github.com/oasislabs/oasis-gateway/auth/core"
	backend "github.com/oasislabs/oasis-gateway/backend/core"
	"github.com/oasislabs/oasis-gateway/errors"
	"github.com/oasislabs/oasis-gateway/log"
	"github.com/oasislabs/oasis-gateway/rpc"
)

// Client interface for the underlying operations needed for the API
// implementation
type Client interface {
	Subscribe(context.Context, backend.SubscribeRequest) (uint64, errors.Err)
	Unsubscribe(context.Context, backend.UnsubscribeRequest) errors.Err
	PollEvent(context.Context, backend.PollEventRequest) (backend.Events, errors.Err)
}

type Services struct {
	Logger log.Logger
	Client Client
}

// EventHandler implements the handlers associated with subscriptions and
// event polling
type EventHandler struct {
	logger log.Logger
	client Client
}

// Subscribe creates a new subscription for the client on the required
// topics
func (h EventHandler) Subscribe(ctx context.Context, v interface{}) (interface{}, error) {
	session := ctx.Value(auth.Session{}).(string)
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

	id, err := h.client.Subscribe(ctx, backend.SubscribeRequest{
		Event:      req.Events[0],
		Address:    query.Get("address"),
		SessionKey: session,
		Topics:     query["topic"],
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
	session := ctx.Value(auth.Session{}).(string)
	req := v.(*UnsubscribeRequest)

	err := h.client.Unsubscribe(ctx, backend.UnsubscribeRequest{
		ID:         req.ID,
		SessionKey: session,
	})
	if err != nil {
		h.logger.Debug(ctx, "failed unsubscribe from events", log.MapFields{
			"call_type": "UnsubscribeFailure",
			"id":        req.ID,
		}, err)
		return nil, err
	}

	return nil, nil
}

// EventPoll allows the user to query for new events associated
// with a specific subscription
func (h EventHandler) PollEvent(ctx context.Context, v interface{}) (interface{}, error) {
	session := ctx.Value(auth.Session{}).(string)
	req := v.(*PollEventRequest)
	if req.Count == 0 {
		req.Count = 10
	}

	res, err := h.client.PollEvent(ctx, backend.PollEventRequest{
		DiscardPrevious: req.DiscardPrevious,
		Count:           req.Count,
		Offset:          req.Offset,
		ID:              req.ID,
		SessionKey:      session,
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
				ID:     r.ID,
				Data:   r.Data,
				Topics: r.Topics,
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

func NewEventHandler(services Services) EventHandler {
	if services.Client == nil {
		panic("Request must be provided as a service")
	}
	if services.Logger == nil {
		panic("Logger must be provided as a service")
	}

	return EventHandler{
		logger: services.Logger.ForClass("event", "handler"),
		client: services.Client,
	}
}

// BindHandler binds the service handler to the provided
// HandlerBinder
func BindHandler(services Services, binder rpc.HandlerBinder) {
	handler := NewEventHandler(services)

	binder.Bind("POST", "/v0/api/event/subscribe", rpc.HandlerFunc(handler.Subscribe),
		rpc.EntityFactoryFunc(func() interface{} { return &SubscribeRequest{} }))
	binder.Bind("POST", "/v0/api/event/unsubscribe", rpc.HandlerFunc(handler.Unsubscribe),
		rpc.EntityFactoryFunc(func() interface{} { return &UnsubscribeRequest{} }))
	binder.Bind("POST", "/v0/api/event/poll", rpc.HandlerFunc(handler.PollEvent),
		rpc.EntityFactoryFunc(func() interface{} { return &PollEventRequest{} }))
}
