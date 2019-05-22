package insecure

import (
	"errors"
	"net/http"

	"github.com/oasislabs/developer-gateway/auth/core"
)

const INSECURE_KEY string = "X-INSECURE-AUTH"

// InsecureAuth is an insecure authentication mechanism that may be
// useful for debugging and testing. It should not be used in
// setups with real users
type InsecureAuth struct{}

func (a InsecureAuth) Authenticate(req *http.Request) (*core.AuthenticationData, error) {
	value := req.Header.Get(INSECURE_KEY)
	if len(value) == 0 {
		return nil, errors.New("Verification failed")
	}
	authData := core.AuthenticationData{
		ExpectedAAD: "",
		SessionKey:  "",
	}
	return &authData, nil
}
