package oauth

import (
	"errors"
	"net/http"

	oidc "github.com/coreos/go-oidc"
	"github.com/oasislabs/developer-gateway/auth/core"
)

const (
	ID_TOKEN_KEY      string = "X-ID-TOKEN"
	googleTokenIssuer string = "https://accounts.google.com"
	googleKeySet      string = "https://www.googleapis.com/oauth2/v3/certs"
)

type GoogleOauth struct{}

type OpenIDClaims struct {
	Email         string `json:"email"`
	EmailVerified bool   `json:"email_verified"`
}

// Authenticates the user using the ID Token receieved from Google.
// Uses the hash of the access token as the session id.
func (g GoogleOauth) Authenticate(req *http.Request) (*core.AuthenticationData, error) {
	rawIDToken := req.Header.Get(ID_TOKEN_KEY)
	keySet := oidc.NewRemoteKeySet(req.Context(), googleKeySet)
	verifier := oidc.NewVerifier(googleTokenIssuer, keySet, &oidc.Config{SkipClientIDCheck: true})

	idToken, err := verifier.Verify(req.Context(), rawIDToken)
	if err != nil {
		return nil, err
	}
	var claims OpenIDClaims
	if err = idToken.Claims(&claims); err != nil {
		return nil, err
	}
	if !claims.EmailVerified {
		return nil, errors.New("Email is unverified")
	}
	authData := core.AuthenticationData{
		ExpectedAAD: claims.Email,
		SessionKey:  idToken.AccessTokenHash,
	}

	return &authData, nil
}
