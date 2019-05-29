package core

import (
	"net/http"
	"testing"

	"github.com/oasislabs/developer-gateway/auth/insecure"
	"github.com/oasislabs/developer-gateway/log"
	"github.com/oasislabs/developer-gateway/rpc"
	"github.com/stretchr/testify/assert"
)

type MockHttpMiddleware struct{}

func (h *MockHttpMiddleware) ServeHTTP(req *http.Request) (interface{}, error) {
	return req.Context().Value(ContextAuthDataKey), nil
}

func TestServeHTTP(t *testing.T) {
	httpMiddlewareAuth := NewHttpMiddlewareAuth(
		insecure.InsecureAuth{},
		log.NewLogrus(log.LogrusLoggerProperties{}),
		&MockHttpMiddleware{})

	req, err := http.NewRequest("POST", "gateway.oasiscloud.io", nil)
	assert.Nil(t, err)
	req.Header.Add(insecure.INSECURE_KEY, "insecure-key")
	req.Header.Add(RequestHeaderSessionKey, "session-key")

	response, err := httpMiddlewareAuth.ServeHTTP(req)
	assert.Nil(t, err)
	authData := response.(AuthData)
	assert.Equal(t, "insecure-key", authData.ExpectedAAD)
	assert.Equal(t, "session-key", authData.SessionKey)
}

func TestServeHTTPNoSessionKey(t *testing.T) {
	httpMiddlewareAuth := NewHttpMiddlewareAuth(
		insecure.InsecureAuth{},
		log.NewLogrus(log.LogrusLoggerProperties{}),
		&MockHttpMiddleware{})

	req, err := http.NewRequest("POST", "gateway.oasiscloud.io", nil)
	assert.Nil(t, err)
	req.Header.Add(insecure.INSECURE_KEY, "insecure-key")

	response, err := httpMiddlewareAuth.ServeHTTP(req)
	assert.NotNil(t, err)
	assert.Equal(t, http.StatusForbidden, err.(*rpc.HttpError).StatusCode)
	assert.Nil(t, response)
}
