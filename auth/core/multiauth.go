package core

import (
	"context"
	stderr "errors"
	"net/http"

	"github.com/oasislabs/developer-gateway/log"
	"github.com/oasislabs/developer-gateway/stats"
)

type MultiAuth struct {
	auths []Auth
}

func (m *MultiAuth) Add(a Auth) {
	m.auths = append(m.auths, a)
}

func (*MultiAuth) Name() string {
	return "auth.MultiAuth"
}
func (m *MultiAuth) Stats() stats.Metrics {
	metrics := make(stats.Metrics)
	for _, auth := range m.auths {
		for k, val := range auth.Stats() {
			metrics[k] = val
		}
	}
	return metrics
}

func (m *MultiAuth) Authenticate(req *http.Request) (*http.Request, error) {
	var errs []error

	for _, auth := range m.auths {
		req, err := auth.Authenticate(req)
		if err != nil {
			errs = append(errs, err)
			continue
		}

		ctx := context.WithValue(req.Context(), m, auth)
		req = req.WithContext(ctx)
		return req, nil
	}

	return req, MultiError{Errors: errs}
}

func (m *MultiAuth) Verify(ctx context.Context, data AuthRequest) error {
	auth := ctx.Value(m)
	if auth == nil {
		return stderr.New("request without auth cannot be verified")
	}

	return auth.(Auth).Verify(ctx, data)
}

func (m *MultiAuth) SetLogger(l log.Logger) {
	for _, auth := range m.auths {
		auth.SetLogger(l)
	}
}
