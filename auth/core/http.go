package core

import (
	"context"
	"crypto/sha1"
	"fmt"
	"net/http"

	"github.com/oasislabs/developer-gateway/errors"
	"github.com/oasislabs/developer-gateway/log"
	"github.com/oasislabs/developer-gateway/rpc"
)

const RequestHeaderSessionKey string = "X-OASIS-SESSION-KEY"

var hasher = sha1.New()

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
	expectedAAD, err := m.auth.Authenticate(req)
	if err != nil {
		return nil, &rpc.HttpError{Cause: nil, StatusCode: http.StatusForbidden}
	}

	sessionKey := req.Header.Get(RequestHeaderSessionKey)
	if len(sessionKey) == 0 {
		return nil, &rpc.HttpError{Cause: nil, StatusCode: http.StatusForbidden}
	}

	if _, err = hasher.Write([]byte(expectedAAD)); err != nil {
		e := errors.New(errors.ErrInvalidAAD, err)
		return nil, &rpc.HttpError{
			Cause:      &e,
			StatusCode: http.StatusForbidden,
		}
	}
	aadHash := string(hasher.Sum(nil))

	authData := AuthData{
		ExpectedAAD: expectedAAD,
		SessionKey:  fmt.Sprintf(sessionKeyFormat, aadHash, sessionKey),
	}

	req = req.WithContext(context.WithValue(req.Context(), ContextAuthDataKey, authData))
	return m.next.ServeHTTP(req)
}
