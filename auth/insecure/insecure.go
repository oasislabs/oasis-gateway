package insecure

import (
	"errors"
	"net/http"

	"github.com/oasislabs/developer-gateway/stats"
)

const HeaderKey string = "X-OASIS-INSECURE-AUTH"

// InsecureAuth is an insecure authentication mechanism that may be
// useful for debugging and testing. It should not be used in
// setups with real users
type InsecureAuth struct{}

func (a InsecureAuth) Name() string {
	return "auth.insecure.InsecureAuth"
}

func (a InsecureAuth) Stats() stats.Metrics {
	return nil
}

func (a InsecureAuth) Authenticate(req *http.Request) (string, error) {
	value := req.Header.Get(HeaderKey)
	if len(value) == 0 {
		return "", errors.New("Verification failed")
	}

	return value, nil
}
