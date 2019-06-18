package oauth

import (
	"context"
	"encoding/binary"
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

const (
	cipherLengthOffset = 16
	aadLengthOffset    = 24
	cipherOffset       = 32
	nonceLength        = 5
)

// Verify the provided AAD in the transaction data with the expected AAD
// Transaction data is expected to be in the following format:
//   pk || cipher length || aad length || cipher || aad || nonce
//   - pk is expected to be 16 bytes
//   - cipher length and aad length are uint64 encoded in big endian
//   - nonce is expected to be 5 bytes
func (GoogleOauth) Verify(data string, expectedAAD string) error {
	if len(data) < cipherOffset {
		return errors.New("Payload data is too short")
	}

	cipherLength := binary.BigEndian.Uint64([]byte(data[cipherLengthOffset:aadLengthOffset]))
	aadLength := binary.BigEndian.Uint64([]byte(data[aadLengthOffset:cipherOffset]))

	if len(data) < int(cipherOffset+cipherLength+aadLength+nonceLength) {
		return errors.New("Missing data")
	}

	aadOffset := cipherOffset + cipherLength
	aad := data[aadOffset : aadOffset+aadLength]

	if aad != expectedAAD {
		return errors.New("AAD does not match")
	}
	return nil
}
