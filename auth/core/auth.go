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
