package core

import (
	"net/http"

	"github.com/oasislabs/developer-gateway/stats"
)

type AuthData struct {
	ExpectedAAD string
	SessionKey  string
}

type Auth interface {
	Name() string
	Stats() stats.Metrics

	// Authenticate the user from the http request. This should return:
	// - the expected AAD
	// - the authentication error
	Authenticate(req *http.Request) (string, error)
	
	// Verify that a specific payload complies with
	// the expected format and has the authentication data required
	Verify(data, expected string) error
}

type NilAuth struct {}
func (NilAuth) Name() string {
	return "auth.nil"
}
func (NilAuth) Stats() stats.Metrics {
	return nil
}
func (NilAuth) Authenticate(req *http.Request) (string, error) {
	return "", nil
}
func (NilAuth) Verify(data, expected string) error {
	return nil
}
