package tests

import (
	"context"
	"reflect"
	"testing"

	"github.com/oasislabs/developer-gateway/eth"
	"github.com/oasislabs/developer-gateway/eth/ethtest"
	"github.com/oasislabs/developer-gateway/tests/apitest"
	"github.com/oasislabs/developer-gateway/tests/gatewaytest"
	"github.com/stretchr/testify/suite"
)

type EventsTestSuite struct {
	suite.Suite
	ethclient     *ethtest.MockClient
	serviceclient *apitest.ServiceClient
	eventclient   *apitest.EventClient
}

func (s *EventsTestSuite) SetupTest() {
	provider, err := gatewaytest.NewServices(context.TODO(), Config)
	if err != nil {
		panic(err)
	}

	s.ethclient = provider.MustGet(reflect.TypeOf((*eth.Client)(nil)).Elem()).(*ethtest.MockClient)

	router := gatewaytest.NewPublicRouter(provider)
	s.eventclient = apitest.NewEventClient(router)
	s.serviceclient = apitest.NewServiceClient(router)
}

func TestEventsTestSuite(t *testing.T) {
	suite.Run(t, new(EventsTestSuite))
}
