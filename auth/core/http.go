package core

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"

	"github.com/oasislabs/developer-gateway/errors"
	"github.com/oasislabs/developer-gateway/log"
	"github.com/oasislabs/developer-gateway/rpc"
)

type AAD struct{}
type Session struct{}

const (
	sessionKeyFormat               = "%s:%s"
	RequestHeaderSessionKey string = "X-OASIS-SESSION-KEY"
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

func MustGetAAD(ctx context.Context) string {
	value := ctx.Value(AAD{})
	if value == nil {
		panic("Authenticate method did not set AAD in http.Request's context")
	}

	return value.(string)
}

func (m *HttpMiddlewareAuth) ServeHTTP(req *http.Request) (interface{}, error) {
	req, err := m.auth.Authenticate(req)
	if err != nil {
		newErr := errors.New(errors.ErrAuthenticateRequest, err)
		return nil, &rpc.HttpError{
			Cause:      &newErr,
			StatusCode: http.StatusForbidden,
		}
	}

	sessionKey := req.Header.Get(RequestHeaderSessionKey)
	if len(sessionKey) == 0 {
		newErr := errors.New(errors.ErrAuthenticateRequest, fmt.Errorf("no %s header provided", RequestHeaderSessionKey))
		return nil, &rpc.HttpError{
			Cause:      &newErr,
			StatusCode: http.StatusForbidden,
		}
	}

	expectedAAD := MustGetAAD(req.Context())
	hasher := sha256.New()
	if _, err = hasher.Write([]byte(expectedAAD)); err != nil {
		return nil, rpc.HttpForbidden(context.TODO(), errors.New(errors.ErrInvalidAAD, err))
	}

	aadHash := hex.EncodeToString(hasher.Sum(nil))

	req = req.WithContext(context.WithValue(req.Context(), Session{}, fmt.Sprintf(sessionKeyFormat, aadHash, sessionKey)))
	return m.next.ServeHTTP(req)
}
