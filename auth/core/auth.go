package core

import "net/http"

type Auth interface {
	Authenticate(req *http.Request) error
}
