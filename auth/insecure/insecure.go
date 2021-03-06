package insecure

import (
	"context"
	"errors"
	"net/http"

	"github.com/oasislabs/oasis-gateway/auth/core"
	"github.com/oasislabs/oasis-gateway/log"
	"github.com/oasislabs/oasis-gateway/stats"
)

const HeaderKey string = "X-OASIS-INSECURE-AUTH"

var ErrDataTooShort = errors.New("Payload data is too short")

// InsecureAuth is an insecure authentication mechanism that may be
// useful for debugging and testing. It should not be used in
// setups with real users.
type InsecureAuth struct{}

func (a InsecureAuth) Name() string {
	return "auth.insecure.InsecureAuth"
}

func (a InsecureAuth) Stats() stats.Metrics {
	return nil
}

func (a InsecureAuth) Authenticate(req *http.Request) (*http.Request, error) {
	value := req.Header.Get(HeaderKey)
	if len(value) == 0 {
		return req, ErrDataTooShort
	}

	ctx := context.WithValue(req.Context(), core.AAD{}, value)
	return req.WithContext(ctx), nil
}

func (InsecureAuth) Verify(ctx context.Context, req core.AuthRequest) error {
	if len(req.Data) == 0 {
		return ErrDataTooShort
	}

	return nil
}

func (InsecureAuth) SetLogger(_ log.Logger) {
}
