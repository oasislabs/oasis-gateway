package core

import (
	"context"
	"net/http"

	"github.com/google/uuid"
	"github.com/oasislabs/developer-gateway/log"
	"github.com/oasislabs/developer-gateway/rpc"
)

const RequestHeaderSessionKey string = "X-OASIS-SESSION-KEY"

type HttpMiddlewareAuth struct {
	auth     Auth
	logger   log.Logger
	next     rpc.HttpMiddleware
	sessions map[string]string // maps session key to owner AAD
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
		auth:     auth,
		logger:   logger.ForClass("auth", "HttpMiddlewareAuth"),
		next:     next,
		sessions: make(map[string]string),
	}
}

func (m *HttpMiddlewareAuth) ServeHTTP(req *http.Request) (interface{}, error) {
	expectedAAD, err := m.auth.Authenticate(req)
	if err != nil {
		return nil, &rpc.HttpError{Cause: nil, StatusCode: http.StatusForbidden}
	}

	sessionKey := req.Header.Get(RequestHeaderSessionKey)
	// Clients request new session by leaving sessionKey empty
	if len(sessionKey) == 0 {
		sessionKey = uuid.New().String()
		for _, ok := m.sessions[sessionKey]; ok; sessionKey = uuid.New().String() {
		}
		m.sessions[sessionKey] = expectedAAD
	}

	if m.sessions[sessionKey] != expectedAAD {
		return nil, &rpc.HttpError{Cause: nil, StatusCode: http.StatusForbidden}
	}

	authData := AuthData{
		ExpectedAAD: expectedAAD,
		SessionKey:  sessionKey,
	}

	req = req.WithContext(context.WithValue(req.Context(), ContextAuthDataKey, authData))
	return m.next.ServeHTTP(req)
}
