package core

import (
	"context"
	"io/ioutil"
	"testing"

	ethereum "github.com/ethereum/go-ethereum/common"
	"github.com/oasislabs/oasis-gateway/errors"
	"github.com/oasislabs/oasis-gateway/log"
	"github.com/oasislabs/oasis-gateway/mqueue/core"
	mqueue "github.com/oasislabs/oasis-gateway/mqueue/core"
	"github.com/oasislabs/oasis-gateway/mqueue/mailboxtest"
	"github.com/oasislabs/oasis-gateway/stats"
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

func (c *MockClient) Senders() []ethereum.Address {
	return []ethereum.Address{
		ethereum.HexToAddress("0x01234567890abcdefa17a5dAfF8dC9b86eE04773"),
		ethereum.HexToAddress("0x0a51514857B379A521C580a10822Fd8A7aC491A0"),
	}
}

func (c *MockClient) GetCode(
	ctx context.Context,
	req GetCodeRequest,
) (GetCodeResponse, errors.Err) {
	args := c.Called(ctx, req)
	if args.Get(1) != nil {
		return GetCodeResponse{}, args.Get(1).(errors.Err)
	}

	return args.Get(0).(GetCodeResponse), nil
}

func (c *MockClient) GetExpiry(
	ctx context.Context,
	req GetExpiryRequest,
) (GetExpiryResponse, errors.Err) {
	args := c.Called(ctx, req)
	if args.Get(1) != nil {
		return GetExpiryResponse{}, args.Get(1).(errors.Err)
	}

	return args.Get(0).(GetExpiryResponse), nil
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

func TestPollEventOKNoDiscard(t *testing.T) {
	manager := createRequestManager()

	manager.mqueue.(*mailboxtest.Mailbox).On("Retrieve",
		mock.Anything, mqueue.RetrieveRequest{
			Key:    "session:sub:0",
			Offset: 0,
			Count:  1,
		}).Return(mqueue.Elements{
		Offset: 0,
		Elements: []core.Element{
			{
				Offset: 0,
				Value:  "{\"ID\": 1, \"Data\": \"value\"}",
				Type:   DataEventType.String(),
			},
		},
	}, nil)

	evs, err := manager.PollEvent(Context, PollEventRequest{
		Offset:          0,
		Count:           1,
		DiscardPrevious: false,
		ID:              0,
		SessionKey:      "session",
	})
	assert.Nil(t, err)
	assert.Equal(t, uint64(0), evs.Offset)
	assert.Equal(t, DataEvent{
		Data:   "value",
		ID:     1,
		Topics: nil,
	}, evs.Events[0])
}

func TestPollEventOKDiscardSubinfo(t *testing.T) {
	manager := createRequestManager()

	manager.mqueue.(*mailboxtest.Mailbox).On("Retrieve",
		mock.Anything, mqueue.RetrieveRequest{
			Key:    "session:sub:0",
			Offset: 0,
			Count:  1,
		}).Return(mqueue.Elements{
		Offset:   0,
		Elements: nil,
	}, nil)
	manager.mqueue.(*mailboxtest.Mailbox).On("Exists",
		mock.Anything, mqueue.ExistsRequest{Key: "session:sub:0"}).
		Return(false, nil)
	manager.mqueue.(*mailboxtest.Mailbox).On("Discard",
		mock.Anything, mqueue.DiscardRequest{
			KeepPrevious: true,
			Count:        1,
			Offset:       0,
			Key:          "session:subinfo",
		}).
		Return(nil)

	evs, err := manager.PollEvent(Context, PollEventRequest{
		Offset:          0,
		Count:           1,
		DiscardPrevious: false,
		ID:              0,
		SessionKey:      "session",
	})
	assert.Nil(t, err)
	assert.Equal(t, uint64(0), evs.Offset)
	assert.Equal(t, 0, len(evs.Events))

	manager.mqueue.(*mailboxtest.Mailbox).AssertCalled(t, "Discard",
		mock.Anything, mqueue.DiscardRequest{
			KeepPrevious: true,
			Count:        1,
			Offset:       0,
			Key:          "session:subinfo",
		})
}
