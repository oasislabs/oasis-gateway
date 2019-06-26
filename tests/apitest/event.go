package apitest

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"time"

	"github.com/google/uuid"
	"github.com/oasislabs/developer-gateway/api/v0/event"
	"github.com/oasislabs/developer-gateway/concurrent"
	"github.com/oasislabs/developer-gateway/rpc"
)

// EventClient is the client implementation for the
// Event API
type EventClient struct {
	client  *Client
	session string
}

// NewEventClient creates a new instance of an event client
// with an underlying client and session ready to be used
// to execute a router API
func NewEventClient(router *rpc.HttpRouter) *EventClient {
	return &EventClient{
		client:  NewClient(router),
		session: uuid.New().String(),
	}
}

// Subscribe creates a subscription to an event topic
func (c *EventClient) Subscribe(
	ctx context.Context,
	req event.SubscribeRequest,
) (event.SubscribeResponse, error) {
	var res event.SubscribeResponse
	if err := c.client.RequestAPI(&rpc.SimpleJsonDeserializer{
		O: &res,
	}, &req, c.session, Route{
		Method: "POST",
		Path:   "/v0/api/event/subscribe",
	}); err != nil {
		return res, err
	}

	return res, nil
}

// Unsubscribe destroys an existing subscription to an event topic
func (c *EventClient) Unsubscribe(
	ctx context.Context,
	req event.UnsubscribeRequest,
) error {
	return c.client.RequestAPI(nil, &req, c.session, Route{
		Method: "POST",
		Path:   "/v0/api/event/unsubscribe",
	})
}

// PollEvent polls for subscription events
func (c *EventClient) PollEvent(
	ctx context.Context,
	req event.PollEventRequest,
) (event.PollEventResponse, error) {
	de := PollEventDataDeserializer{}

	if err := c.client.RequestAPI(&de, &req, c.session, Route{
		Method: "POST",
		Path:   "/v0/api/event/poll",
	}); err != nil {
		return event.PollEventResponse{}, err
	}

	return event.PollEventResponse{
		Offset: de.Offset,
		Events: de.Events,
	}, nil
}

// PollEventUntilNotEmpty polls for events until at least
// one event is received or it times out
func (c *EventClient) PollEventUntilNotEmpty(
	ctx context.Context,
	req event.PollEventRequest,
) (event.PollEventResponse, error) {
	v, err := concurrent.RetryWithConfig(ctx, concurrent.SupplierFunc(func() (interface{}, error) {
		v, err := c.PollEvent(ctx, req)
		if err != nil {
			return nil, concurrent.ErrCannotRecover{Cause: err}
		}

		if len(v.Events) == 0 {
			return nil, errors.New("no events yet")
		}

		return v, nil
	}), concurrent.RetryConfig{
		Random:            false,
		UnlimitedAttempts: false,
		Attempts:          10,
		BaseExp:           2,
		BaseTimeout:       1 * time.Millisecond,
		MaxRetryTimeout:   100 * time.Millisecond,
	})

	if err != nil {
		return event.PollEventResponse{}, err
	}

	return v.(event.PollEventResponse), nil
}

type PollEventDataDeserialized struct {
	Offset uint64            `json:"offset"`
	Events []event.DataEvent `json:"events"`
}

type PollEventDataDeserializer struct {
	Events []event.Event
	Offset uint64
}

func (d *PollEventDataDeserializer) Deserialize(r io.Reader) error {
	var res PollEventDataDeserialized
	if err := json.NewDecoder(r).Decode(&res); err != nil {
		return err
	}

	var events []event.Event
	for _, ev := range res.Events {
		events = append(events, ev)
	}

	d.Events = events
	d.Offset = res.Offset
	return nil
}
