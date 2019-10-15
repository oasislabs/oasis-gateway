package insecure

import (
	"context"
	"net/http"
	"testing"

	"github.com/oasislabs/oasis-gateway/auth/core"
	"github.com/oasislabs/oasis-gateway/log"
	"github.com/oasislabs/oasis-gateway/rpc"
	"github.com/stretchr/testify/assert"
)

type MockHTTPMiddleware struct{}

func (h *MockHTTPMiddleware) ServeHTTP(req *http.Request) (interface{}, error) {
	return req.Context(), nil
}

func TestServeHTTP(t *testing.T) {
	httpMiddlewareAuth := core.NewHttpMiddlewareAuth(
		InsecureAuth{},
		log.NewLogrus(log.LogrusLoggerProperties{}),
		&MockHTTPMiddleware{})

	req, err := http.NewRequest("POST", "gateway.oasiscloud.io", nil)
	assert.Nil(t, err)
	req.Header.Add(HeaderKey, "insecure-key")
	req.Header.Add(core.RequestHeaderSessionKey, "session-key")

	v, err := httpMiddlewareAuth.ServeHTTP(req)
	assert.Nil(t, err)

	ctx := v.(context.Context)
	aad := ctx.Value(core.AAD{})
	session := ctx.Value(core.Session{})
	assert.Equal(t, "017fdef9eeec58e0ad6b94721a2eb52a9bd96dddd9aa2f1e058153568f4ed42d:session-key", session)
	assert.Equal(t, "insecure-key", aad)
}

func TestServeHTTPInvalidSessionKey(t *testing.T) {
	httpMiddlewareAuth := core.NewHttpMiddlewareAuth(
		InsecureAuth{},
		log.NewLogrus(log.LogrusLoggerProperties{}),
		&MockHTTPMiddleware{})

	req, err := http.NewRequest("POST", "gateway.oasiscloud.io", nil)
	assert.Nil(t, err)
	req.Header.Add(HeaderKey, "insecure-key")

	response, err := httpMiddlewareAuth.ServeHTTP(req)
	assert.NotNil(t, err)
	assert.Equal(t, http.StatusForbidden, err.(*rpc.HttpError).StatusCode)
	assert.Nil(t, response)
}

func TestServeHTTPNonMatchingSessionKeys(t *testing.T) {
	httpMiddlewareAuth := core.NewHttpMiddlewareAuth(
		InsecureAuth{},
		log.NewLogrus(log.LogrusLoggerProperties{}),
		&MockHTTPMiddleware{})

	req1, err := http.NewRequest("POST", "gateway.oasiscloud.io", nil)
	assert.Nil(t, err)
	req1.Header.Add(HeaderKey, "user-1")
	req1.Header.Add(core.RequestHeaderSessionKey, "session-key")

	v, err := httpMiddlewareAuth.ServeHTTP(req1)
	assert.Nil(t, err)

	ctx := v.(context.Context)
	session1 := ctx.Value(core.Session{})

	req2, err := http.NewRequest("POST", "gateway.oasiscloud.io", nil)
	assert.Nil(t, err)
	req2.Header.Add(HeaderKey, "user-2")
	req2.Header.Add(core.RequestHeaderSessionKey, "session-key")

	v, err = httpMiddlewareAuth.ServeHTTP(req2)
	assert.Nil(t, err)

	ctx = v.(context.Context)
	session2 := ctx.Value(core.Session{})

	assert.NotEqual(t, session1, session2)
}
