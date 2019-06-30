package core

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"testing"

	"github.com/oasislabs/developer-gateway/stats"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// ExtAuthHeader is the header in the *http.Request that contains
// the relevant authentication data for the server
const ExtAuthHeader = "X-EXT-AUTH"

type Client interface {
	Do(*http.Request) (*http.Response, error)
}

// ExtAuthVerifier authenticates an *http.Request and verifies
// that the issuer has the right permissions to execute
// the requested API against an external authentication server
type ExtAuthVerifier struct {
	successes stats.Counter
	failures  stats.Counter
	client    Client
}

func NewExtAuthVerifier(client Client) *ExtAuthVerifier {
	return &ExtAuthVerifier{
		client: client,
	}
}

// ExtAuthPayload is the payload sent out to the authentication
// server for request authentication
type ExtAuthPayload struct {
	// RequestData is the data provided by the auth framework to
	// check for the request legitimacy to be executed
	RequestData AuthRequest

	// Token is the data in the header of the *http.Request
	// collected in the call to ExtAuthVerifier.Authenticate
	Token string
}

func (v *ExtAuthVerifier) Name() string {
	return "ExtAuthVerifier"
}

func (v *ExtAuthVerifier) Stats() stats.Metrics {
	return stats.Metrics{
		"successes": v.successes.Value(),
		"failures":  v.failures.Value(),
	}
}

// Authenticate returns the contents of the ExtAuthHeader on success
func (v *ExtAuthVerifier) Authenticate(req *http.Request) (string, error) {
	value := req.Header.Get(ExtAuthHeader)
	if len(value) == 0 {
		v.failures.Incr()
		return "", fmt.Errorf("missing request header %s", ExtAuthHeader)
	}

	return value, nil
}

// Verify makes a request to the external authentication server. This
// method succeeds based on the resposne
func (v *ExtAuthVerifier) Verify(req AuthRequest, token string) error {
	buffer := bytes.NewBuffer(nil)
	if err := json.NewEncoder(buffer).Encode(ExtAuthPayload{
		RequestData: req,
		Token:       token,
	}); err != nil {
		return err
	}

	httpReq, err := http.NewRequest("POST", "/authenticate", buffer)
	if err != nil {
		return err
	}

	res, err := v.client.Do(httpReq)
	if err != nil {
		return err
	}

	if res.StatusCode != http.StatusOK && res.StatusCode != http.StatusNoContent {
		return errors.New("request not authorized")
	}

	return nil
}

type HttpClientMock struct {
	mock.Mock
}

func (m *HttpClientMock) Do(req *http.Request) (*http.Response, error) {
	args := m.Called(req)
	if args.Get(1) != nil {
		return nil, args.Error(1)
	}

	return args.Get(0).(*http.Response), nil
}

func TestExtAuthVerifyName(t *testing.T) {
	assert.Equal(t, "ExtAuthVerifier", (&ExtAuthVerifier{}).Name())
}

func TestExtAuthVerifyInitialStats(t *testing.T) {
	assert.Equal(t, stats.Metrics{
		"successes": uint64(0),
		"failures":  uint64(0),
	}, (&ExtAuthVerifier{}).Stats())
}

func TestExtAuthAuthenticateMissingHeader(t *testing.T) {
	verifier := NewExtAuthVerifier(&HttpClientMock{})

	req, _ := http.NewRequest("GET", "URL", nil)

	_, err := verifier.Authenticate(req)
	assert.Equal(t, "missing request header X-EXT-AUTH", err.Error())
}

func TestExtAuthAuthenticateAndVerifyOK(t *testing.T) {
	client := &HttpClientMock{}
	verifier := NewExtAuthVerifier(client)
	req, _ := http.NewRequest("GET", "URL", nil)
	req.Header.Add(ExtAuthHeader, "token")

	res := &http.Response{StatusCode: http.StatusOK}
	client.On("Do", mock.Anything).Return(res, nil)

	v, err := verifier.Authenticate(req)
	assert.Nil(t, err)

	err = verifier.Verify(AuthRequest{
		API:     "MyAPI",
		Address: "",
		AAD:     "John Doe",
		Data:    "some data",
	}, v)
	assert.Nil(t, err)
}

func TestExtAuthAuthenticateAndVerifyErr(t *testing.T) {
	client := &HttpClientMock{}
	verifier := NewExtAuthVerifier(client)
	req, _ := http.NewRequest("GET", "URL", nil)
	req.Header.Add(ExtAuthHeader, "token")

	res := &http.Response{StatusCode: http.StatusForbidden}
	client.On("Do", mock.Anything).Return(res, nil)

	v, err := verifier.Authenticate(req)
	assert.Nil(t, err)

	err = verifier.Verify(AuthRequest{
		API:     "MyAPI",
		Address: "",
		AAD:     "John Doe",
		Data:    "some data",
	}, v)
	assert.Equal(t, "request not authorized", err.Error())
}
