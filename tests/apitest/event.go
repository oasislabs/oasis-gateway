package apitest

import (
	"context"

	"github.com/google/uuid"
	"github.com/oasislabs/developer-gateway/api/v0/event"
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
	var res event.PollEventResponse
	if err := c.client.RequestAPI(&rpc.SimpleJsonDeserializer{
		O: &res,
	}, &req, c.session, Route{
		Method: "POST",
		Path:   "/v0/api/event/poll",
	}); err != nil {
		return res, err
	}

	return res, nil
}
