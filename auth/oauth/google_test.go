package oauth

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/oasislabs/developer-gateway/auth/core"
	"github.com/stretchr/testify/assert"
)

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
	req, err = auth.Authenticate(req)
	assert.Nil(t, err)
	assert.Equal(t, "test@email.com", req.Context().Value(core.AAD{}))
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
	req, err = auth.Authenticate(req)
	assert.NotNil(t, err)
	assert.Equal(t, req, req)
	assert.Equal(t, "Email is unverified", err.Error())
	assert.Nil(t, req.Context().Value(core.AAD{}))
}
