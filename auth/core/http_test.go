package core

import (
	"net/http"
	"testing"

	"github.com/oasislabs/developer-gateway/auth/insecure"
	"github.com/oasislabs/developer-gateway/log"
	"github.com/oasislabs/developer-gateway/rpc"
	"github.com/stretchr/testify/assert"
)

type MockHTTPMiddleware struct{}

func (h *MockHTTPMiddleware) ServeHTTP(req *http.Request) (interface{}, error) {
	return req.Context().Value(ContextAuthDataKey), nil
}

func TestServeHTTP(t *testing.T) {
	httpMiddlewareAuth := NewHttpMiddlewareAuth(
		insecure.InsecureAuth{},
		log.NewLogrus(log.LogrusLoggerProperties{}),
		&MockHTTPMiddleware{})

	req, err := http.NewRequest("POST", "gateway.oasiscloud.io", nil)
	assert.Nil(t, err)
	req.Header.Add(insecure.HeaderKey, "insecure-key")
	req.Header.Add(RequestHeaderSessionKey, "session-key")

	response, err := httpMiddlewareAuth.ServeHTTP(req)
	assert.Nil(t, err)
	authData := response.(AuthData)
	assert.Equal(t, "017fdef9eeec58e0ad6b94721a2eb52a9bd96dddd9aa2f1e058153568f4ed42d:session-key", authData.SessionKey)
	assert.Equal(t, "insecure-key", authData.ExpectedAAD)
	assert.NotNil(t, authData.SessionKey)
}

func TestServeHTTPInvalidSessionKey(t *testing.T) {
	httpMiddlewareAuth := NewHttpMiddlewareAuth(
		insecure.InsecureAuth{},
		log.NewLogrus(log.LogrusLoggerProperties{}),
		&MockHTTPMiddleware{})

	req, err := http.NewRequest("POST", "gateway.oasiscloud.io", nil)
	assert.Nil(t, err)
	req.Header.Add(insecure.HeaderKey, "insecure-key")

	response, err := httpMiddlewareAuth.ServeHTTP(req)
	assert.NotNil(t, err)
	assert.Equal(t, http.StatusForbidden, err.(*rpc.HttpError).StatusCode)
	assert.Nil(t, response)
}

func TestServeHTTPNonMatchingSessionKeys(t *testing.T) {
	httpMiddlewareAuth := NewHttpMiddlewareAuth(
		insecure.InsecureAuth{},
		log.NewLogrus(log.LogrusLoggerProperties{}),
		&MockHTTPMiddleware{})

	req1, err := http.NewRequest("POST", "gateway.oasiscloud.io", nil)
	assert.Nil(t, err)
	req1.Header.Add(insecure.HeaderKey, "user-1")
	req1.Header.Add(RequestHeaderSessionKey, "session-key")

	response1, err := httpMiddlewareAuth.ServeHTTP(req1)
	assert.Nil(t, err)
	authData1 := response1.(AuthData)

	req2, err := http.NewRequest("POST", "gateway.oasiscloud.io", nil)
	assert.Nil(t, err)
	req2.Header.Add(insecure.HeaderKey, "user-2")
	req2.Header.Add(RequestHeaderSessionKey, "session-key")

	response2, err := httpMiddlewareAuth.ServeHTTP(req2)
	assert.Nil(t, err)
	authData2 := response2.(AuthData)

	assert.NotEqual(t, authData1.SessionKey, authData2.SessionKey)
}
