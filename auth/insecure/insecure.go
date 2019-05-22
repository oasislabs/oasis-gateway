package insecure

import (
	"net/http"
)

// InsecureAuth is an insecure authentication mechanism that may be
// useful for debugging and testing. It should not be used in
// setups with real users
type InsecureAuth struct{}

func (a InsecureAuth) Authenticate(req *http.Request) error {
	return nil
}
