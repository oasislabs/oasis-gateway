package tests

import (
	"context"
	"reflect"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/oasislabs/developer-gateway/api/v0/event"
	"github.com/oasislabs/developer-gateway/concurrent"
	"github.com/oasislabs/developer-gateway/eth"
	"github.com/oasislabs/developer-gateway/eth/ethtest"
	"github.com/oasislabs/developer-gateway/rpc"
	"github.com/oasislabs/developer-gateway/tests/apitest"
	"github.com/oasislabs/developer-gateway/tests/gatewaytest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

type EventsTestSuite struct {
	suite.Suite
	ethclient   *ethtest.MockClient
	eventclient *apitest.EventClient
}

func (s *EventsTestSuite) SetupTest() {
	provider, err := gatewaytest.NewServices(context.TODO(), Config)
	if err != nil {
		panic(err)
	}

	s.ethclient = provider.MustGet(reflect.TypeOf((*eth.Client)(nil)).Elem()).(*ethtest.MockClient)

	router := gatewaytest.NewPublicRouter(provider)
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

	evs, err := s.eventclient.PollEventUntilNotEmpty(context.TODO(), event.PollEventRequest{
		ID:     0,
		Offset: 0,
		Count:  1,
	})
	assert.Nil(s.T(), err)
	assert.Equal(s.T(), event.PollEventResponse{
		Offset: 0x0,
		Events: []event.Event{
			event.DataEvent{ID: 0x0, Data: "0x"},
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

	err = s.eventclient.Unsubscribe(context.TODO(), event.UnsubscribeRequest{
		ID: 0,
	})
	assert.Nil(s.T(), err)

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
