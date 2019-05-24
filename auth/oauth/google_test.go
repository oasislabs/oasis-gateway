package oauth

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

const CTX_ID_TOKEN = "id_token"

type MockIDToken struct {
	claims []byte
}

func (mock *MockIDToken) Claims(v interface{}) error {
	return json.Unmarshal(mock.claims, v)
}

type MockIDTokenVerifier struct{}

func (mock *MockIDTokenVerifier) Verify(ctx context.Context, rawIDToken string) (IDToken, error) {
	return &MockIDToken{claims: []byte(rawIDToken)}, nil
}

func TestAuthenticateSuccess(t *testing.T) {
	claims := OpenIDClaims{
		Email:         "test@email.com",
		EmailVerified: true,
	}
	jsonStr, err := json.Marshal(claims)
	assert.Nil(t, err)

	req, err := http.NewRequest("POST", "gateway.oasiscloud.io", nil)
	assert.Nil(t, err)
	req.Header.Add(ID_TOKEN_KEY, string(jsonStr))

	auth := NewGoogleOauth(&MockIDTokenVerifier{})
	email, err := auth.Authenticate(req)
	assert.Nil(t, err)
	assert.Equal(t, email, "test@email.com")
}

func TestAuthenticateUnverified(t *testing.T) {
	claims := OpenIDClaims{
		Email:         "test@email.com",
		EmailVerified: false,
	}
	jsonStr, err := json.Marshal(claims)
	assert.Nil(t, err)

	req, err := http.NewRequest("POST", "gateway.oasiscloud.io", nil)
	assert.Nil(t, err)
	req.Header.Add(ID_TOKEN_KEY, string(jsonStr))

	auth := NewGoogleOauth(&MockIDTokenVerifier{})
	email, err := auth.Authenticate(req)
	assert.NotNil(t, err)
	assert.Equal(t, err.Error(), "Email is unverified")
	assert.Equal(t, email, "")
}
