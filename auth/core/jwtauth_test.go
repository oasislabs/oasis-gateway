package core

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"testing"
	"time"

	jwt "github.com/dgrijalva/jwt-go"
	"github.com/oasislabs/oasis-gateway/stats"
	"github.com/stretchr/testify/assert"
)

// JwtHeader is the header in the *http.Request that contains
// the JWT
const JwtHeader = "X-JWT-AUTH"

// JwtVerifier authenticates an *http.Request and verifies
// that the issuer has the right permissions to execute
// the requested API
type JwtVerifier struct {
	successes stats.Counter
	failures  stats.Counter
}

func (v *JwtVerifier) Name() string {
	return "JwtVerifier"
}

func (v *JwtVerifier) Stats() stats.Metrics {
	return stats.Metrics{
		"successes": v.successes.Value(),
		"failures":  v.failures.Value(),
	}
}

// JwtData represents the relevant authentication data
// from the *http.Request that needs to be verified
type JwtData struct {
	// Scope is the scope defined as part of the
	// JWT claims
	Scope string `json:"scope"`

	// Name is the name of the user as part of the
	// JWT claims
	Name string `json:"name"`
}

// Authenticate returns a json encoded JwtData object on success
// with the relevant data for the verification.
func (v *JwtVerifier) Authenticate(req *http.Request) (string, error) {
	value := req.Header.Get(JwtHeader)
	if len(value) == 0 {
		v.failures.Incr()
		return "", fmt.Errorf("missing request header %s", JwtHeader)
	}

	// here we authenticate the token by verifying that the signature
	// is correct
	t, err := jwt.Parse(value, func(token *jwt.Token) (interface{}, error) {
		return []byte("secret"), nil
	})
	if err != nil {
		v.failures.Incr()
		return "", err
	}

	// collect the relevant data from the token into JwtData
	// and serialize it into JSON to be latter processed
	// by Verify
	p, err := json.Marshal(JwtData{
		Scope: t.Claims.(jwt.MapClaims)["scope"].(string),
		Name:  t.Claims.(jwt.MapClaims)["name"].(string),
	})
	if err != nil {
		return "", err
	}

	return string(p), err
}

// Verify that the data in an encoded JwtData matches the
// verifier expectations and the request can proceed the
// normal flow
func (v *JwtVerifier) Verify(req AuthRequest, encoded string) error {
	var data JwtData
	if err := json.Unmarshal([]byte(encoded), &data); err != nil {
		v.failures.Incr()
		return err
	}

	if data.Name != string(req.AAD) {
		v.failures.Incr()
		return errors.New("request AAD does not match token identity name")
	}

	if data.Scope != req.API {
		v.failures.Incr()
		return errors.New("request API does not match token scope")
	}

	v.successes.Incr()
	return nil
}

type Claims struct {
	jwt.StandardClaims
	Scope string `json:"scope"`
	Name  string `json:"name"`
}

func generateToken() (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, &Claims{
		Scope: "MyAPI",
		Name:  "John Doe",
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: time.Now().Unix() + 1000,
		},
	})

	return token.SignedString([]byte("secret"))
}

func TestJwtAuthVerifyName(t *testing.T) {
	assert.Equal(t, "JwtVerifier", (&JwtVerifier{}).Name())
}

func TestJwtAuthVerifyInitialStats(t *testing.T) {
	assert.Equal(t, stats.Metrics{
		"successes": uint64(0),
		"failures":  uint64(0),
	}, (&JwtVerifier{}).Stats())
}

func TestJwtAuthAuthenticateMissingHeader(t *testing.T) {
	req, _ := http.NewRequest("GET", "URL", nil)

	_, err := (&JwtVerifier{}).Authenticate(req)
	assert.Equal(t, "missing request header X-JWT-AUTH", err.Error())
}

func TestJwtAuthAuthenticateAndVerifyOK(t *testing.T) {
	token, err := generateToken()
	assert.Nil(t, err)
	req, _ := http.NewRequest("GET", "URL", nil)
	req.Header.Add(JwtHeader, token)

	verifier := &JwtVerifier{}
	v, err := verifier.Authenticate(req)
	assert.Nil(t, err)

	err = verifier.Verify(AuthRequest{
		API:     "MyAPI",
		Address: "",
		AAD:     []byte("John Doe"),
		Data:    "some data",
	}, v)
	assert.Nil(t, err)
}

func TestJwtAuthAuthenticateAndVerifyErrIdentity(t *testing.T) {
	token, err := generateToken()
	assert.Nil(t, err)
	req, _ := http.NewRequest("GET", "URL", nil)
	req.Header.Add(JwtHeader, token)

	verifier := &JwtVerifier{}
	v, err := verifier.Authenticate(req)
	assert.Nil(t, err)

	err = verifier.Verify(AuthRequest{
		API:     "MyAPI",
		Address: "",
		AAD:     []byte("My Name"),
		Data:    "some data",
	}, v)
	assert.Equal(t, "request AAD does not match token identity name", err.Error())
}

func TestJwtAuthAuthenticateAndVerifyErrScope(t *testing.T) {
	token, err := generateToken()
	assert.Nil(t, err)
	req, _ := http.NewRequest("GET", "URL", nil)
	req.Header.Add(JwtHeader, token)

	verifier := &JwtVerifier{}
	v, err := verifier.Authenticate(req)
	assert.Nil(t, err)

	err = verifier.Verify(AuthRequest{
		API:     "MyAPI",
		Address: "",
		AAD:     []byte("My Name"),
		Data:    "some data",
	}, v)
	assert.Equal(t, "request AAD does not match token identity name", err.Error())
}
