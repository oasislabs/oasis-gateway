package tests

import (
	"net/http"
	"testing"

	auth "github.com/oasislabs/developer-gateway/auth/core"
	"github.com/oasislabs/developer-gateway/auth/insecure"
	"github.com/oasislabs/developer-gateway/tests/apitest"
	"github.com/stretchr/testify/assert"
)

func TestPathNotAuth(t *testing.T) {
	res, err := apitest.NewClient(router).Request(apitest.Request{
		Route: apitest.Route{
			Method: "POST",
			Path:   "/v0/api/service/deploy",
		},
		Body:    nil,
		Headers: nil,
	})

	assert.Nil(t, err)
	assert.Equal(t, http.StatusForbidden, res.Code)
	assert.Equal(t, "", string(res.Body))
}

func TestPathNoSession(t *testing.T) {
	res, err := apitest.NewClient(router).Request(apitest.Request{
		Route: apitest.Route{
			Method: "POST",
			Path:   "/v0/api/service/deploy",
		},
		Body: nil,
		Headers: map[string]string{
			insecure.HeaderKey: "mykey",
		},
	})
	assert.Nil(t, err)

	assert.Equal(t, http.StatusForbidden, res.Code)
	assert.Equal(t, "", string(res.Body))
}

func TestPathUnknownPath(t *testing.T) {
	res, err := apitest.NewClient(router).Request(apitest.Request{
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
	assert.Nil(t, err)

	assert.Equal(t, http.StatusNotFound, res.Code)
	assert.Equal(t, "", string(res.Body))
}

func TestPathInvalidMethod(t *testing.T) {
	res, err := apitest.NewClient(router).Request(apitest.Request{
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
	assert.Nil(t, err)

	assert.Equal(t, http.StatusMethodNotAllowed, res.Code)
	assert.Equal(t, "", string(res.Body))
}

func TestPathNoContentType(t *testing.T) {
	res, err := apitest.NewClient(router).Request(apitest.Request{
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
	assert.Nil(t, err)

	assert.Equal(t, http.StatusBadRequest, res.Code)
	assert.Equal(t, "{\"errorCode\":2004,\"description\":\"Content-type should be application/json.\"}\n", string(res.Body))
}
