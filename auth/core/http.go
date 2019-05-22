package core

import (
	"context"
	"net/http"

	"github.com/oasislabs/developer-gateway/log"
	"github.com/oasislabs/developer-gateway/rpc"
)

type HttpMiddlewareAuth struct {
	auth   Auth
	logger log.Logger
	next   rpc.HttpMiddleware
}

func NewHttpMiddlewareAuth(auth Auth, logger log.Logger, next rpc.HttpMiddleware) *HttpMiddlewareAuth {
	if auth == nil {
		panic("auth must be set")
	}

	if logger == nil {
		panic("log must be set")
	}

	if next == nil {
		panic("next must be set")
	}

	return &HttpMiddlewareAuth{
		auth:   auth,
		logger: logger.ForClass("auth", "HttpMiddlewareAuth"),
		next:   next,
	}
}

func (m *HttpMiddlewareAuth) ServeHTTP(req *http.Request) (interface{}, error) {
	authData, err := m.auth.Authenticate(req)

	if err != nil {
		return nil, &rpc.HttpError{Cause: nil, StatusCode: http.StatusForbidden}
	}

	req = req.WithContext(context.WithValue(req.Context(), ContextExpectedAADKey, authData.ExpectedAAD))
	req = req.WithContext(context.WithValue(req.Context(), ContextSessionKey, authData.SessionKey))
	return m.next.ServeHTTP(req)
}
