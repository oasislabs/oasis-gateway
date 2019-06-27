package event

import (
	"context"
	"io/ioutil"
	"testing"

	auth "github.com/oasislabs/developer-gateway/auth/core"
	backend "github.com/oasislabs/developer-gateway/backend/core"
	"github.com/oasislabs/developer-gateway/errors"
	"github.com/oasislabs/developer-gateway/log"
	"github.com/oasislabs/developer-gateway/rpc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

var Context = context.TODO()

var Logger = log.NewLogrus(log.LogrusLoggerProperties{
	Output: ioutil.Discard,
})

type MockClient struct {
	mock.Mock
}

func (c *MockClient) Subscribe(
	ctx context.Context,
	req backend.SubscribeRequest,
) (uint64, errors.Err) {
	args := c.Called(ctx, req)
	if args.Get(1) != nil {
		return 0, args.Get(1).(errors.Err)
	}

	return args.Get(0).(uint64), nil
}

func (c *MockClient) Unsubscribe(
	ctx context.Context,
	req backend.UnsubscribeRequest,
) errors.Err {
	args := c.Called(ctx, req)
	if args.Get(1) != nil {
		return args.Get(1).(errors.Err)
	}

	return nil
}

func (c *MockClient) PollEvent(
	ctx context.Context,
	req backend.PollEventRequest,
) (backend.Events, errors.Err) {
	args := c.Called(ctx, req)
	if args.Get(1) != nil {
		return backend.Events{}, args.Get(1).(errors.Err)
	}

	return args.Get(0).(backend.Events), nil
}

type InvalidEvent struct{}

func (e InvalidEvent) EventID() uint64 {
	return 0
}

func (e InvalidEvent) EventType() backend.EventType {
	return backend.DataEventType
}

func createEventHandler() EventHandler {
	return NewEventHandler(Services{
		Logger: Logger,
		Client: &MockClient{},
	})
}

func TestSubscribeErrNoEvents(t *testing.T) {
	ctx := context.WithValue(Context, auth.ContextAuthDataKey, auth.AuthData{
		ExpectedAAD: "",
		SessionKey:  "sessionKey",
	})

	handler := createEventHandler()

	_, err := handler.Subscribe(ctx, &SubscribeRequest{
		Events: nil,
		Filter: "",
	})

	assert.Equal(t, "[2007] error code InputError with desc Input cannot be empty. with cause no events set on request", err.Error())
}

func TestSubscribeErrTooManyEvents(t *testing.T) {
	ctx := context.WithValue(Context, auth.ContextAuthDataKey, auth.AuthData{
		ExpectedAAD: "",
		SessionKey:  "sessionKey",
	})

	handler := createEventHandler()

	_, err := handler.Subscribe(ctx, &SubscribeRequest{
		Events: []string{"event1", "event2"},
		Filter: "",
	})

	assert.Equal(t, "[2007] error code InputError with desc Input cannot be empty. with cause only one event supported", err.Error())
}

func TestSubscribeErrInvalidQueryParams(t *testing.T) {
	ctx := context.WithValue(Context, auth.ContextAuthDataKey, auth.AuthData{
		ExpectedAAD: "",
		SessionKey:  "sessionKey",
	})

	handler := createEventHandler()

	_, err := handler.Subscribe(ctx, &SubscribeRequest{
		Events: []string{"event1"},
		Filter: "this is not a query",
	})

	assert.Equal(t, "[2009] error code InputError with desc Failed to parse query parameters.", err.Error())
}

func TestSubscribeErrReturn(t *testing.T) {
	ctx := context.WithValue(Context, auth.ContextAuthDataKey, auth.AuthData{
		ExpectedAAD: "",
		SessionKey:  "sessionKey",
	})

	handler := createEventHandler()

	handler.client.(*MockClient).On("Subscribe", mock.Anything, mock.Anything).
		Return(0, errors.New(errors.ErrInternalError, nil))

	_, err := handler.Subscribe(ctx, &SubscribeRequest{
		Events: []string{"event"},
		Filter: "address=myaddress",
	})

	assert.Equal(t, "[1000] error code InternalError with desc Internal Error. Please check the status of the service.", err.Error())
}

func TestSubscribeOKWithTopics(t *testing.T) {
	ctx := context.WithValue(Context, auth.ContextAuthDataKey, auth.AuthData{
		ExpectedAAD: "",
		SessionKey:  "sessionKey",
	})

	handler := createEventHandler()

	handler.client.(*MockClient).On("Subscribe", mock.Anything, mock.Anything).
		Return(uint64(1), nil)

	res, err := handler.Subscribe(ctx, &SubscribeRequest{
		Events: []string{"event"},
		Filter: "address=myaddress&topic=topic1&topic=topic2",
	})

	assert.Nil(t, err)
	assert.Equal(t, SubscribeResponse{
		ID: 1,
	}, res)
	handler.client.(*MockClient).AssertCalled(t, "Subscribe", ctx, backend.SubscribeRequest{
		Event:      "event",
		Address:    "myaddress",
		SessionKey: "sessionKey",
		Topics:     []string{"topic1", "topic2"},
	})
}

func TestSubscribeOKNoTopic(t *testing.T) {
	ctx := context.WithValue(Context, auth.ContextAuthDataKey, auth.AuthData{
		ExpectedAAD: "",
		SessionKey:  "sessionKey",
	})

	handler := createEventHandler()

	handler.client.(*MockClient).On("Subscribe", mock.Anything, mock.Anything).
		Return(uint64(1), nil)

	res, err := handler.Subscribe(ctx, &SubscribeRequest{
		Events: []string{"event"},
		Filter: "address=myaddress",
	})

	assert.Nil(t, err)
	assert.Equal(t, SubscribeResponse{
		ID: 1,
	}, res)
	handler.client.(*MockClient).AssertCalled(t, "Subscribe", ctx, backend.SubscribeRequest{
		Event:      "event",
		Address:    "myaddress",
		SessionKey: "sessionKey",
		Topics:     nil,
	})
}

func TestUnsubscribeOK(t *testing.T) {
	ctx := context.WithValue(Context, auth.ContextAuthDataKey, auth.AuthData{
		ExpectedAAD: "",
		SessionKey:  "sessionKey",
	})

	handler := createEventHandler()

	handler.client.(*MockClient).On("Unsubscribe", mock.Anything, mock.Anything).
		Return(uint64(1), nil)

	res, err := handler.Unsubscribe(ctx, &UnsubscribeRequest{
		ID: 0,
	})

	assert.Nil(t, err)
	assert.Nil(t, res)
}

func TestUnsubscribeErrReturn(t *testing.T) {
	ctx := context.WithValue(Context, auth.ContextAuthDataKey, auth.AuthData{
		ExpectedAAD: "",
		SessionKey:  "sessionKey",
	})

	handler := createEventHandler()

	handler.client.(*MockClient).On("Unsubscribe", mock.Anything, mock.Anything).
		Return(0, errors.New(errors.ErrInternalError, nil))

	_, err := handler.Unsubscribe(ctx, &UnsubscribeRequest{
		ID: 0,
	})

	assert.Equal(t, "[1000] error code InternalError with desc Internal Error. Please check the status of the service.", err.Error())
}

func TestPollEventOKEmpty(t *testing.T) {
	ctx := context.WithValue(Context, auth.ContextAuthDataKey, auth.AuthData{
		ExpectedAAD: "",
		SessionKey:  "sessionKey",
	})

	handler := createEventHandler()

	handler.client.(*MockClient).On("PollEvent", mock.Anything, mock.Anything).
		Return(backend.Events{}, nil)

	res, err := handler.PollEvent(ctx, &PollEventRequest{
		Offset: 0,
	})

	assert.Nil(t, err)
	assert.Equal(t, PollEventResponse{
		Offset: 0,
		Events: []Event{},
	}, res)
}

func TestPollEventOKMultiple(t *testing.T) {
	ctx := context.WithValue(Context, auth.ContextAuthDataKey, auth.AuthData{
		ExpectedAAD: "",
		SessionKey:  "sessionKey",
	})

	handler := createEventHandler()

	handler.client.(*MockClient).On("PollEvent", mock.Anything, mock.Anything).
		Return(backend.Events{
			Offset: 0,
			Events: []backend.Event{
				backend.DataEvent{
					ID:   0,
					Data: "0x000000",
				},
				backend.ErrorEvent{
					ID:    1,
					Cause: rpc.Error{},
				},
			}}, nil)

	res, err := handler.PollEvent(ctx, &PollEventRequest{
		Offset: 0,
	})

	assert.Nil(t, err)
	assert.Equal(t, PollEventResponse{
		Offset: 0,
		Events: []Event{
			DataEvent{
				ID:   0,
				Data: "0x000000"},
			ErrorEvent{
				ID:    1,
				Cause: rpc.Error{},
			},
		}}, res)
}

func TestPollEventErrUnknown(t *testing.T) {
	ctx := context.WithValue(Context, auth.ContextAuthDataKey, auth.AuthData{
		ExpectedAAD: "",
		SessionKey:  "sessionKey",
	})

	handler := createEventHandler()

	handler.client.(*MockClient).On("PollEvent", mock.Anything, mock.Anything).
		Return(backend.Events{
			Offset: 0,
			Events: []backend.Event{InvalidEvent{}}}, nil)

	assert.Panics(t, func() {
		_, _ = handler.PollEvent(ctx, &PollEventRequest{
			Offset: 0,
		})
	})
}

func TestPollEventErrReturn(t *testing.T) {
	ctx := context.WithValue(Context, auth.ContextAuthDataKey, auth.AuthData{
		ExpectedAAD: "",
		SessionKey:  "sessionKey",
	})

	handler := createEventHandler()

	handler.client.(*MockClient).On("PollEvent", mock.Anything, mock.Anything).
		Return(backend.Events{
			Offset: 0,
			Events: nil,
		}, errors.New(errors.ErrInternalError, nil))

	_, err := handler.PollEvent(ctx, &PollEventRequest{
		Offset: 0,
	})

	assert.Error(t, err)
}

func TestNewEventHandlerNoClient(t *testing.T) {
	assert.Panics(t, func() {
		NewEventHandler(Services{
			Client: nil,
			Logger: Logger,
		})
	})
}

func TestNewEventHandlerNoLogger(t *testing.T) {
	assert.Panics(t, func() {
		NewEventHandler(Services{
			Client: &MockClient{},
			Logger: nil,
		})
	})
}

func TestNewEventHandlerOK(t *testing.T) {
	h := NewEventHandler(Services{
		Client: &MockClient{},
		Logger: Logger,
	})

	assert.NotNil(t, h)
}

func TestBindHandlerOK(t *testing.T) {
	binder := rpc.NewHttpBinder(rpc.HttpBinderProperties{
		Encoder: rpc.JsonEncoder{},
		Logger:  Logger,
		HandlerFactory: rpc.HttpHandlerFactoryFunc(func(factory rpc.EntityFactory, handler rpc.Handler) rpc.HttpMiddleware {
			return rpc.NewHttpJsonHandler(rpc.HttpJsonHandlerProperties{
				Limit:   1 << 16,
				Handler: handler,
				Logger:  Logger,
				Factory: factory,
			})
		}),
	})

	BindHandler(Services{
		Client: &MockClient{},
		Logger: Logger,
	}, binder)

	router := binder.Build()

	assert.True(t, router.HasHandler("/v0/api/event/subscribe", "POST"))
	assert.True(t, router.HasHandler("/v0/api/event/unsubscribe", "POST"))
	assert.True(t, router.HasHandler("/v0/api/event/poll", "POST"))
}
