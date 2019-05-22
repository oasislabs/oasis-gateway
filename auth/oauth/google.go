package oauth

import (
	"net/http"

	oidc "github.com/coreos/go-oidc"
)

const (
	ID_TOKEN_KEY      string = "X-ID-TOKEN"
	googleUserInfo    string = "https://openidconnect.googleapis.com/v1/userinfo"
	googleTokenIssuer string = "https://accounts.google.com"
	googleKeySet      string = "https://www.googleapis.com/oauth2/v3/certs"
)

type GoogleOauth struct{}

func (g GoogleOauth) Authenticate(req *http.Request) error {
	rawIDToken := req.Header.Get(ID_TOKEN_KEY)
	keySet := oidc.NewRemoteKeySet(req.Context(), googleKeySet)
	verifier := oidc.NewVerifier(googleTokenIssuer, keySet, oidc.Config{SkipClientIDCheck: true})

	idToken, err := verifier.Verify(rawIDToken)
	if err != nil {
		return err
	}
	var claims struct {
		Email         string `json:"email"`
		EmailVerified bool   `json:"email_verified"`
	}

	if err = idToken.Claims(&claims); err != nil {
		return err
	}
}
