package tests

import (
	"context"
	"reflect"
	"testing"

	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/oasislabs/oasis-gateway/api/v0/event"
	backend "github.com/oasislabs/oasis-gateway/backend/core"
	"github.com/oasislabs/oasis-gateway/concurrent"
	"github.com/oasislabs/oasis-gateway/eth"
	"github.com/oasislabs/oasis-gateway/eth/ethtest"
	"github.com/oasislabs/oasis-gateway/rpc"
	"github.com/oasislabs/oasis-gateway/stats"
	"github.com/oasislabs/oasis-gateway/tests/apitest"
	"github.com/oasislabs/oasis-gateway/tests/gatewaytest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

type EventsTestSuite struct {
	suite.Suite
	ethclient   *ethtest.MockClient
	eventclient *apitest.EventClient
	request     *backend.RequestManager
}

func (s *EventsTestSuite) SetupTest() {
	provider, err := gatewaytest.NewServices(context.TODO(), Config)
	if err != nil {
		panic(err)
	}

	s.ethclient = provider.MustGet(reflect.TypeOf((*eth.Client)(nil)).Elem()).(*ethtest.MockClient)
	s.request = provider.MustGet(reflect.TypeOf((&backend.RequestManager{}))).(*backend.RequestManager)

	router := gatewaytest.NewPublicRouter(Config, provider)
	s.eventclient = apitest.NewEventClient(router)
}

func (s *EventsTestSuite) TestSubscribeErrEvent() {
	_, err := s.eventclient.Subscribe(context.TODO(), event.SubscribeRequest{
		Events: []string{"invalid"},
		Filter: "address=address",
	})

	assert.Equal(s.T(),
		&rpc.Error{
			ErrorCode:   2012,
			Description: "Only logs topic supported for subscriptions.",
		}, err)
}

func (s *EventsTestSuite) TestSubscribeOK() {
	sub := &ethtest.MockSubscription{ErrC: make(chan error, 1)}

	ethtest.ImplementMockWithOverwrite(s.ethclient,
		ethtest.MockMethods{
			"SubscribeFilterLogs": ethtest.MockMethod{
				Arguments: []interface{}{mock.Anything, mock.Anything, mock.Anything},
				Return:    []interface{}{sub, nil},
				Run: func(args mock.Arguments) {
					c := args.Get(2).(chan<- types.Log)
					c <- types.Log{
						Address:     common.HexToAddress("0x0000000000000000000000000000000000000000"),
						BlockNumber: 1,
						Topics: []common.Hash{
							common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000000"),
							common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000001"),
						},
					}
				},
			},
		})

	res, err := s.eventclient.Subscribe(context.TODO(), event.SubscribeRequest{
		Events: []string{"logs"},
		Filter: "address=address&topic=0x0000000000000000000000000000000000000000000000000000000000000000&topic=0x0000000000000000000000000000000000000000000000000000000000000001",
	})
	assert.Nil(s.T(), err)
	assert.Equal(s.T(), event.SubscribeResponse{
		ID: 0,
	}, res)

	s.ethclient.AssertCalled(s.T(), "SubscribeFilterLogs",
		mock.Anything, ethereum.FilterQuery{
			Addresses: []common.Address{common.HexToAddress("address")},
			Topics: [][]common.Hash{
				{common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000000")},
				{common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000001")},
			},
		}, mock.Anything)

	evs, err := s.eventclient.PollEventUntilNotEmpty(context.TODO(), event.PollEventRequest{
		ID:     0,
		Offset: 0,
		Count:  1,
	})
	assert.Nil(s.T(), err)
	assert.Equal(s.T(), event.PollEventResponse{
		Offset: 0x0,
		Events: []event.Event{
			event.DataEvent{
				ID:   0x0,
				Data: "0x",
				Topics: []string{
					"0x0000000000000000000000000000000000000000000000000000000000000000",
					"0x0000000000000000000000000000000000000000000000000000000000000001",
				},
			},
		}}, evs)
}

func (s *EventsTestSuite) TestUnsubscribeErrNoExists() {
	ethtest.ImplementMock(s.ethclient)

	err := s.eventclient.Unsubscribe(context.TODO(), event.UnsubscribeRequest{
		ID: 0,
	})

	assert.Equal(s.T(), &rpc.Error{
		ErrorCode:   6002,
		Description: "Subscription not found.",
	}, err)
}

func (s *EventsTestSuite) TestUnsubscribeOK() {
	sub := &ethtest.MockSubscription{ErrC: make(chan error, 1)}

	ethtest.ImplementMockWithOverwrite(s.ethclient,
		ethtest.MockMethods{
			"SubscribeFilterLogs": ethtest.MockMethod{
				Arguments: []interface{}{mock.Anything, mock.Anything, mock.Anything},
				Return:    []interface{}{sub, nil},
				Run: func(args mock.Arguments) {
					c := args.Get(2).(chan<- types.Log)
					c <- types.Log{
						Address:     common.HexToAddress("0x0000000000000000000000000000000000000000"),
						BlockNumber: 1,
					}
				},
			},
		})

	res, err := s.eventclient.Subscribe(context.TODO(), event.SubscribeRequest{
		Events: []string{"logs"},
		Filter: "address=address",
	})
	assert.Nil(s.T(), err)
	assert.Equal(s.T(), event.SubscribeResponse{
		ID: 0,
	}, res)
	subStats := s.request.Stats()["subscriptions"].(stats.Metrics)
	assert.Equal(s.T(), uint64(1), subStats["subscriptionCount"])
	assert.Equal(s.T(), uint64(1), subStats["currentSubscriptions"])
	assert.Equal(s.T(), uint64(1), subStats["totalSubscriptionCount"])

	err = s.eventclient.Unsubscribe(context.TODO(), event.UnsubscribeRequest{
		ID: 0,
	})
	assert.Nil(s.T(), err)
	subStats = s.request.Stats()["subscriptions"].(stats.Metrics)
	assert.Equal(s.T(), uint64(0), subStats["subscriptionCount"])
	assert.Equal(s.T(), uint64(0), subStats["currentSubscriptions"])
	assert.Equal(s.T(), uint64(1), subStats["totalSubscriptionCount"])

	_, err = s.eventclient.PollEventUntilNotEmpty(context.TODO(), event.PollEventRequest{
		ID:     0,
		Offset: 0,
		Count:  1,
	})

	causes, ok := err.(concurrent.ErrMaxAttemptsReached)
	assert.True(s.T(), ok)
	for i := 0; i < len(causes.Causes); i++ {
		assert.Equal(s.T(), "no events yet", causes.Causes[i].Error())
	}
}

func TestEventsTestSuite(t *testing.T) {
	suite.Run(t, new(EventsTestSuite))
}
