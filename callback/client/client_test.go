package client

import (
	"context"
	"io/ioutil"
	"net/http"
	"testing"
	"time"

	"github.com/oasislabs/developer-gateway/conc"
	"github.com/oasislabs/developer-gateway/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

var Context = context.TODO()

var TestRetryConfig = conc.RetryConfig{
	BaseTimeout:     1,
	BaseExp:         1,
	MaxRetryTimeout: 10 * time.Millisecond,
	Attempts:        10,
	Random:          false,
}

var Logger = log.NewLogrus(log.LogrusLoggerProperties{
	Output: ioutil.Discard,
})

type MockHttpClient struct {
	mock.Mock
}

func (c *MockHttpClient) Do(req *http.Request) (*http.Response, error) {
	args := c.Called(req)
	if args.Get(1) != nil {
		return nil, args.Error(1)
	}

	return args.Get(0).(*http.Response), nil
}

func newClient() *Client {
	return NewClientWithDeps(&Deps{
		Client: &MockHttpClient{},
		Logger: Logger,
	}, &Props{
		Callbacks:   Callbacks{},
		RetryConfig: TestRetryConfig,
	})
}

func TestClientCallbackDisabledNoSend(t *testing.T) {
	client := newClient()

	err := client.callback(Context, &Callback{Enabled: false})

	assert.Nil(t, err)
	client.client.(*MockHttpClient).AssertNotCalled(t, "Do", mock.Anything)
}

func TestClientCallbackSendOK(t *testing.T) {
	client := newClient()

	client.client.(*MockHttpClient).On("Do", mock.MatchedBy(func(req *http.Request) bool {
		return req.Method == http.MethodPost &&
			req.URL.String() == "http://localhost:1234/" &&
			req.Header.Get("Content-type") == "plain/text"
	})).Return(&http.Response{StatusCode: http.StatusOK}, nil)

	err := client.callback(Context, &Callback{
		Enabled: true,
		Method:  http.MethodPost,
		URL:     "http://localhost:1234/",
		Body:    "some body",
		Headers: []string{"Content-type:plain/text"},
	})

	assert.Nil(t, err)
}

func TestClientCallbackSendNotOK(t *testing.T) {
	client := newClient()

	client.client.(*MockHttpClient).On("Do", mock.MatchedBy(func(req *http.Request) bool {
		return req.Method == http.MethodPost &&
			req.URL.String() == "http://localhost:1234/" &&
			req.Header.Get("Content-type") == "plain/text"
	})).Return(&http.Response{StatusCode: http.StatusInternalServerError}, nil)

	err := client.callback(Context, &Callback{
		Enabled: true,
		Method:  http.MethodPost,
		URL:     "http://localhost:1234/",
		Body:    "some body",
		Headers: []string{"Content-type:plain/text"},
	})

	assert.Error(t, err)
	client.client.(*MockHttpClient).AssertCalled(t, "Do", mock.Anything)
}
