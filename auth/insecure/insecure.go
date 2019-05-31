package insecure

import (
	"errors"
	"net/http"
)

const HeaderKey string = "X-OASIS-INSECURE-AUTH"

// InsecureAuth is an insecure authentication mechanism that may be
// useful for debugging and testing. It should not be used in
// setups with real users
type InsecureAuth struct{}

func (a InsecureAuth) Authenticate(req *http.Request) (string, error) {
	value := req.Header.Get(HeaderKey)
	if len(value) == 0 {
		return "", errors.New("Verification failed")
	}

	return value, nil
}
