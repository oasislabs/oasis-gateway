package oauth

import (
	"context"
	"errors"
	"net/http"

	oidc "github.com/coreos/go-oidc"
	"github.com/oasislabs/developer-gateway/stats"
)

const (
	ID_TOKEN_KEY      string = "X-ID-TOKEN"
	googleTokenIssuer string = "https://accounts.google.com"
	googleKeySet      string = "https://www.googleapis.com/oauth2/v3/certs"
)

type IDToken interface {
	Claims(v interface{}) error
}

type IDTokenVerifier interface {
	Verify(ctx context.Context, rawIDToken string) (IDToken, error)
}

type GoogleIDTokenVerifier struct {
	verifier *oidc.IDTokenVerifier
}

func (g *GoogleIDTokenVerifier) Verify(ctx context.Context, rawIDToken string) (IDToken, error) {
	return g.verifier.Verify(ctx, rawIDToken)
}

type GoogleOauth struct {
	verifier IDTokenVerifier
}

type OpenIDClaims struct {
	Email         string `json:"email"`
	EmailVerified bool   `json:"email_verified"`
}

func NewGoogleOauth(verifier IDTokenVerifier) GoogleOauth {
	return GoogleOauth{verifier: verifier}
}

func (g GoogleOauth) Name() string {
	return "auth.oauth.GoogleOauth"
}

func (g GoogleOauth) Stats() stats.Metrics {
	return nil
}

// Authenticates the user using the ID Token receieved from Google.
func (g GoogleOauth) Authenticate(req *http.Request) (string, error) {
	rawIDToken := req.Header.Get(ID_TOKEN_KEY)
	verifier := g.verifier
	if verifier == nil {
		keySet := oidc.NewRemoteKeySet(req.Context(), googleKeySet)
		verifier = &GoogleIDTokenVerifier{
			verifier: oidc.NewVerifier(googleTokenIssuer, keySet, &oidc.Config{SkipClientIDCheck: true}),
		}
	}

	idToken, err := verifier.Verify(req.Context(), rawIDToken)
	if err != nil {
		return "", err
	}
	var claims OpenIDClaims
	if err = idToken.Claims(&claims); err != nil {
		return "", err
	}
	if !claims.EmailVerified {
		return "", errors.New("Email is unverified")
	}

	return claims.Email, nil
}
