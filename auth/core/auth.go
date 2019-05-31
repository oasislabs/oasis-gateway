package core

import "net/http"

type AuthData struct {
	ExpectedAAD string
	SessionKey  string
}

type Auth interface {
	// Authenticate the user from the http request. This should return:
	// - the expected AAD
	// - the authentication error
	Authenticate(req *http.Request) (string, error)
}

// Verifier to verify that a specific payload complies with
// the expected format and has the authentication data required
type Verifier interface {
	Verify(data, expected string) error
}
