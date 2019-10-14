package core

import (
	"context"
	"fmt"
	"net/http"

	"github.com/oasislabs/oasis-gateway/log"
	"github.com/oasislabs/oasis-gateway/stats"
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

func (a *NilAuth) Name() string {
	return "auth.nil"
}
func (a *NilAuth) Stats() stats.Metrics {
	return nil
}
func (a *NilAuth) Authenticate(req *http.Request) (*http.Request, error) {
	ctx := context.WithValue(req.Context(), AAD{}, "nil")
	return req.WithContext(ctx), nil
}
func (a *NilAuth) Verify(ctx context.Context, req AuthRequest) error {
	fmt.Println(ctx, ctx.Value(AAD{}))
	if "nil" == ctx.Value(AAD{}).(string) {
		return nil
	}

	panic("request authenticated by NilAuth does not have nil as AAD")
}

func (a *NilAuth) SetLogger(log.Logger) {
}
