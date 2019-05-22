package core

import "net/http"

type Auth interface {
	// Authenticate the user from the http request. This should return:
	// - the expected AAD
	// - the session key
	// - the authentication error
	Authenticate(req *http.Request) (string, string, error)
}
