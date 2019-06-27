package core

import (
	"context"
	"io/ioutil"
	"testing"

	"github.com/oasislabs/developer-gateway/errors"
	"github.com/oasislabs/developer-gateway/log"
	mqueue "github.com/oasislabs/developer-gateway/mqueue/core"
	"github.com/oasislabs/developer-gateway/mqueue/mailboxtest"
	"github.com/oasislabs/developer-gateway/stats"
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

func (c *MockClient) Name() string {
	return "backend.core.MockClient"
}

func (c *MockClient) Stats() stats.Metrics {
	return nil
}

func (c *MockClient) GetPublicKey(
	ctx context.Context,
	req GetPublicKeyRequest,
) (GetPublicKeyResponse, errors.Err) {
	args := c.Called(ctx, req)
	if args.Get(1) != nil {
		return GetPublicKeyResponse{}, args.Get(1).(errors.Err)
	}

	return args.Get(0).(GetPublicKeyResponse), nil
}

func (c *MockClient) ExecuteService(
	ctx context.Context,
	id uint64,
	req ExecuteServiceRequest,
) (ExecuteServiceResponse, errors.Err) {
	args := c.Called(ctx, id, req)
	if args.Get(1) != nil {
		return ExecuteServiceResponse{}, args.Get(1).(errors.Err)
	}

	return args.Get(0).(ExecuteServiceResponse), nil
}

func (c *MockClient) DeployService(
	ctx context.Context,
	id uint64,
	req DeployServiceRequest,
) (DeployServiceResponse, errors.Err) {
	args := c.Called(ctx, id, req)
	if args.Get(1) != nil {
		return DeployServiceResponse{}, args.Get(1).(errors.Err)
	}

	return args.Get(0).(DeployServiceResponse), nil
}

func (c *MockClient) SubscribeRequest(
	ctx context.Context,
	req CreateSubscriptionRequest,
	ch chan<- interface{},
) errors.Err {
	args := c.Called(ctx, req, ch)
	if args.Get(0) != nil {
		return args.Get(0).(errors.Err)
	}

	return nil
}

func (c *MockClient) UnsubscribeRequest(
	ctx context.Context,
	req DestroySubscriptionRequest,
) errors.Err {
	args := c.Called(ctx, req)
	if args.Get(0) != nil {
		return args.Get(0).(errors.Err)
	}

	return nil
}

func createRequestManager() *RequestManager {
	return NewRequestManager(RequestManagerProperties{
		MQueue: &mailboxtest.Mailbox{},
		Client: &MockClient{},
		Logger: Logger,
	})

}

func TestSubscribeErrNoSessionKey(t *testing.T) {
	manager := createRequestManager()

	_, err := manager.Subscribe(Context, SubscribeRequest{
		Event:   "event",
		Address: "address",
		Topics:  []string{"topic1", "topic2"},
	})

	assert.Equal(t, "[2011] error code InputError with desc Provided invalid key. with cause key cannot be empty", err.Error())
}

func TestSubscribeOK(t *testing.T) {
	manager := createRequestManager()

	manager.mqueue.(*mailboxtest.Mailbox).On("Next",
		mock.Anything, mock.Anything).Return(uint64(0), nil)

	manager.client.(*MockClient).On("SubscribeRequest",
		mock.Anything, mock.Anything, mock.Anything).Return(nil)

	id, err := manager.Subscribe(Context, SubscribeRequest{
		Event:      "event",
		Address:    "address",
		SessionKey: "session",
		Topics:     []string{"topic1", "topic2"},
	})

	assert.Nil(t, err)
	assert.Equal(t, uint64(0), id)

	manager.mqueue.(*mailboxtest.Mailbox).AssertCalled(t, "Next",
		mock.Anything, mqueue.NextRequest{
			Key: "session:subinfo",
		})
	manager.client.(*MockClient).AssertCalled(t, "SubscribeRequest",
		mock.Anything, CreateSubscriptionRequest{
			Event:   "event",
			Address: "address",
			SubID:   "session:sub:0",
			Topics:  []string{"topic1", "topic2"},
		}, mock.Anything)
}
