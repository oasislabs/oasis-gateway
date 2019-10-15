package tests

import (
	"context"
	"net/http"
	"reflect"
	"testing"

	auth "github.com/oasislabs/oasis-gateway/auth/core"
	"github.com/oasislabs/oasis-gateway/auth/insecure"
	"github.com/oasislabs/oasis-gateway/eth"
	"github.com/oasislabs/oasis-gateway/eth/ethtest"
	"github.com/oasislabs/oasis-gateway/tests/apitest"
	"github.com/oasislabs/oasis-gateway/tests/gatewaytest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type ApiTestSuite struct {
	suite.Suite
	client *apitest.Client
}

func (s *ApiTestSuite) SetupTest() {
	provider, err := gatewaytest.NewServices(context.TODO(), Config)
	if err != nil {
		panic(err)
	}

	ethclient := provider.MustGet(reflect.TypeOf((*eth.Client)(nil)).Elem()).(*ethtest.MockClient)
	ethtest.ImplementMock(ethclient)

	router := gatewaytest.NewPublicRouter(Config, provider)
	s.client = apitest.NewClient(router)
}

func (s *ApiTestSuite) TestPathNotAuth() {
	res, err := s.client.Request(apitest.Request{
		Route: apitest.Route{
			Method: "POST",
			Path:   "/v0/api/service/deploy",
		},
		Body:    nil,
		Headers: nil,
	})

	assert.Nil(s.T(), err)
	assert.Equal(s.T(), http.StatusForbidden, res.Code)
	assert.Equal(s.T(), "{\"errorCode\":7003,\"description\":\"Failed to authenticate request.\"}\n", string(res.Body))
}

func (s *ApiTestSuite) TestPathNoSession() {
	res, err := s.client.Request(apitest.Request{
		Route: apitest.Route{
			Method: "POST",
			Path:   "/v0/api/service/deploy",
		},
		Body: nil,
		Headers: map[string]string{
			insecure.HeaderKey: "mykey",
		},
	})
	assert.Nil(s.T(), err)

	assert.Equal(s.T(), http.StatusForbidden, res.Code)
	assert.Equal(s.T(), "{\"errorCode\":7003,\"description\":\"Failed to authenticate request.\"}\n", string(res.Body))
}

func (s *ApiTestSuite) TestPathUnknownPath() {
	res, err := s.client.Request(apitest.Request{
		Route: apitest.Route{
			Method: "POST",
			Path:   "/v0/api/service/unknown",
		},
		Body: nil,
		Headers: map[string]string{
			insecure.HeaderKey:           "mykey",
			auth.RequestHeaderSessionKey: "mysession",
		},
	})
	assert.Nil(s.T(), err)

	assert.Equal(s.T(), http.StatusNotFound, res.Code)
	assert.Equal(s.T(), "", string(res.Body))
}

func (s *ApiTestSuite) TestPathInvalidMethod() {
	res, err := s.client.Request(apitest.Request{
		Route: apitest.Route{
			Method: "GET",
			Path:   "/v0/api/service/deploy",
		},
		Body: nil,
		Headers: map[string]string{
			insecure.HeaderKey:           "mykey",
			auth.RequestHeaderSessionKey: "mysession",
		},
	})
	assert.Nil(s.T(), err)

	assert.Equal(s.T(), http.StatusMethodNotAllowed, res.Code)
	assert.Equal(s.T(), "", string(res.Body))
}

func (s *ApiTestSuite) TestPathNoContentType() {
	res, err := s.client.Request(apitest.Request{
		Route: apitest.Route{
			Method: "POST",
			Path:   "/v0/api/service/deploy",
		},
		Body: []byte("{}"),
		Headers: map[string]string{
			insecure.HeaderKey:           "mykey",
			auth.RequestHeaderSessionKey: "mysession",
			"Content-length":             "2",
		},
	})
	assert.Nil(s.T(), err)

	assert.Equal(s.T(), http.StatusBadRequest, res.Code)
	assert.Equal(s.T(), "{\"errorCode\":2004,\"description\":\"Content-type should be application/json.\"}\n", string(res.Body))
}

func TestApiTestSuite(t *testing.T) {
	suite.Run(t, new(ApiTestSuite))
}
