package core

import (
	"context"
	"net/http"

	"github.com/oasislabs/developer-gateway/log"
	"github.com/oasislabs/developer-gateway/stats"
)

type AuthRequest struct {
	API     string
	Address string
	AAD     []byte
	PK      []byte
	Data    string
}

type Auth interface {
	Name() string
	Stats() stats.Metrics

	// Authenticate the user from the http request. This should return:
	// - the expected AAD
	// - the authentication error
	Authenticate(req *http.Request) (*http.Request, error)

	// Verify that a specific payload complies with
	// the expected format and has the authentication data required
	Verify(ctx context.Context, req AuthRequest) error

	// Sets the logger for the authentication plugin.
	SetLogger(log.Logger)
}

type NilAuth struct{}

func (NilAuth) Name() string {
	return "auth.nil"
}
func (NilAuth) Stats() stats.Metrics {
	return nil
}
func (NilAuth) Authenticate(req *http.Request) (*http.Request, error) {
	return req, nil
}
func (NilAuth) Verify(ctx context.Context, req AuthRequest) error {
	return nil
}

func (NilAuth) SetLogger(log.Logger) {
}
