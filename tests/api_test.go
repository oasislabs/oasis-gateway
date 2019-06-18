package tests

import (
	"context"
	"net/http"
	"testing"

	auth "github.com/oasislabs/developer-gateway/auth/core"
	"github.com/oasislabs/developer-gateway/auth/insecure"
	"github.com/oasislabs/developer-gateway/gateway"
	"github.com/oasislabs/developer-gateway/tests/apitest"
	"github.com/oasislabs/developer-gateway/tests/mock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type ApiTestSuite struct {
	suite.Suite
	client *apitest.Client
}

func (s *ApiTestSuite) SetupTest() {
	services, err := mock.NewServices(context.TODO(), Config)
	if err != nil {
		panic(err)
	}

	router := gateway.NewPublicRouter(services)
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
	assert.Equal(s.T(), "", string(res.Body))
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
	assert.Equal(s.T(), "", string(res.Body))
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
