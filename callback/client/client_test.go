package client

import (
	"context"
	"html/template"
	"io/ioutil"
	"math/big"
	"net/http"
	"testing"
	"time"

	"github.com/oasislabs/developer-gateway/concurrent"
	"github.com/oasislabs/developer-gateway/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

var Context = context.TODO()

var TestRetryConfig = concurrent.RetryConfig{
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
	mockclient := client.client.(*MockHttpClient)

	err := client.Callback(Context,
		&Callback{Enabled: false, Sync: true},
		&CallbackProps{})

	assert.Nil(t, err)
	mockclient.AssertNotCalled(t, "Do", mock.Anything)
}

func TestClientCallbackSendOK(t *testing.T) {
	client := newClient()
	mockclient := client.client.(*MockHttpClient)

	mockclient.On("Do",
		mock.MatchedBy(func(req *http.Request) bool {
			return req.Method == http.MethodPost &&
				req.URL.String() == "http://localhost:1234/" &&
				req.Header.Get("Content-type") == "plain/text"
		})).Return(&http.Response{StatusCode: http.StatusOK}, nil)

	err := client.Callback(Context, &Callback{
		Enabled:    true,
		Method:     http.MethodPost,
		URL:        "http://localhost:1234/",
		BodyFormat: nil,
		Headers:    []string{"Content-type:plain/text"},
		Sync:       true,
	}, &CallbackProps{})

	assert.Nil(t, err)
	mockclient.AssertCalled(t, "Do", mock.Anything)
}

func TestClientCallbackSendNotOK(t *testing.T) {
	client := newClient()
	mockclient := client.client.(*MockHttpClient)

	mockclient.On("Do",
		mock.MatchedBy(func(req *http.Request) bool {
			return req.Method == http.MethodPost &&
				req.URL.String() == "http://localhost:1234/" &&
				req.Header.Get("Content-type") == "plain/text"
		})).Return(&http.Response{StatusCode: http.StatusInternalServerError}, nil)

	err := client.Callback(Context, &Callback{
		Enabled:    true,
		Method:     http.MethodPost,
		URL:        "http://localhost:1234/",
		BodyFormat: nil,
		Headers:    []string{"Content-type:plain/text"},
		Sync:       true,
	}, &CallbackProps{})

	_, ok := err.(concurrent.ErrMaxAttemptsReached)
	assert.True(t, ok)
	mockclient.AssertCalled(t, "Do", mock.Anything)
}

func TestClientWalletOutOfFundsOK(t *testing.T) {
	bodyTmpl, err := template.New("WalletOutOfFundsBody").Parse("{\"address\": \"{{.Address}}\"}")
	assert.Nil(t, err)

	queryURLTmpl, err := template.New("WalletOutOfFundsQueryURL").Parse("address={{.Address}}")
	assert.Nil(t, err)

	client := newClient()
	mockclient := client.client.(*MockHttpClient)

	mockclient.On("Do", mock.Anything).
		Return(&http.Response{StatusCode: http.StatusOK}, nil)

	err = client.Callback(Context, &Callback{
		Enabled:        true,
		Method:         http.MethodPost,
		URL:            "http://localhost:1234/",
		BodyFormat:     bodyTmpl,
		QueryURLFormat: queryURLTmpl,
		Headers:        []string{"Content-type:plain/text"},
		Sync:           true,
	}, &CallbackProps{Body: WalletOutOfFundsBody{
		Address: "myAddress",
	}})

	assert.Nil(t, err)
	mockclient.AssertCalled(t, "Do", mock.MatchedBy(func(req *http.Request) bool {
		v, err := ioutil.ReadAll(req.Body)
		if err != nil {
			return false
		}

		return string(v) == "{\"address\": \"myAddress\"}" &&
			req.URL.RawQuery == "address=myAddress"
	}))
}

func TestClientWalletReachedFundsThresholdOKCalledBeforeSet(t *testing.T) {
	bodyTmplFmt := `{
  "address": "{{.Address}}",
  "before": "{{.Before}}",
  "after": "{{.After}}",
  "threshold": "{{.Threshold}}"
}`
	queryTmplFmt := "address={{.Address}}&before={{.Before}}" +
		"&after={{.After}}&threshold={{.Threshold}}"
	bodyTmpl, err := template.New("WalletReachedFundsThreshold").Parse(bodyTmplFmt)
	assert.Nil(t, err)

	queryURLTmpl, err := template.New("WalletReachedFundsThreshold").Parse(queryTmplFmt)
	assert.Nil(t, err)

	client := newClient()
	mockclient := client.client.(*MockHttpClient)
	client.callbacks = Callbacks{
		WalletReachedFundsThreshold: WalletReachedFundsThresholdCallback{
			Callback: Callback{
				Enabled:        true,
				Method:         http.MethodPost,
				URL:            "http://localhost:1234/",
				BodyFormat:     bodyTmpl,
				QueryURLFormat: queryURLTmpl,
				Headers:        []string{"Content-type:plain/text"},
				Sync:           true,
			},
			Threshold: new(big.Int).SetInt64(1024),
		},
	}

	mockclient.On("Do", mock.Anything).
		Return(&http.Response{StatusCode: http.StatusOK}, nil)

	client.WalletReachedFundsThreshold(context.TODO(), WalletReachedFundsThresholdBody{
		Before:  new(big.Int).SetInt64(1025),
		After:   new(big.Int).SetInt64(1023),
		Address: "0x0000000000000000000000000000000000000000",
	})

	assert.Nil(t, err)
	mockclient.AssertCalled(t, "Do", mock.MatchedBy(func(req *http.Request) bool {
		v, err := ioutil.ReadAll(req.Body)
		if err != nil {
			return false
		}

		return string(v) == `{
  "address": "0x0000000000000000000000000000000000000000",
  "before": "0x401",
  "after": "0x3ff",
  "threshold": "0x400"
}` && req.URL.RawQuery == "address=0x0000000000000000000000000000000000000000"+
			"&before=0x401&after=0x3ff&threshold=0x400"
	}))
}

func TestClientWalletReachedFundsThresholdOKCalledBeforeNil(t *testing.T) {
	bodyTmplFmt := `{
  "address": "{{.Address}}",
  "before": "{{.Before}}",
  "after": "{{.After}}",
  "threshold": "{{.Threshold}}"
}`
	queryTmplFmt := "address={{.Address}}&before={{.Before}}" +
		"&after={{.After}}&threshold={{.Threshold}}"
	bodyTmpl, err := template.New("WalletReachedFundsThreshold").Parse(bodyTmplFmt)
	assert.Nil(t, err)

	queryURLTmpl, err := template.New("WalletReachedFundsThreshold").Parse(queryTmplFmt)
	assert.Nil(t, err)

	client := newClient()
	mockclient := client.client.(*MockHttpClient)
	client.callbacks = Callbacks{
		WalletReachedFundsThreshold: WalletReachedFundsThresholdCallback{
			Callback: Callback{
				Enabled:        true,
				Method:         http.MethodPost,
				URL:            "http://localhost:1234/",
				BodyFormat:     bodyTmpl,
				QueryURLFormat: queryURLTmpl,
				Headers:        []string{"Content-type:plain/text"},
				Sync:           true,
			},
			Threshold: new(big.Int).SetInt64(1024),
		},
	}

	mockclient.On("Do", mock.Anything).
		Return(&http.Response{StatusCode: http.StatusOK}, nil)

	client.WalletReachedFundsThreshold(context.TODO(), WalletReachedFundsThresholdBody{
		Before:  nil,
		After:   new(big.Int).SetInt64(1023),
		Address: "0x0000000000000000000000000000000000000000",
	})

	assert.Nil(t, err)
	mockclient.AssertCalled(t, "Do", mock.MatchedBy(func(req *http.Request) bool {
		v, err := ioutil.ReadAll(req.Body)
		if err != nil {
			return false
		}

		return string(v) == `{
  "address": "0x0000000000000000000000000000000000000000",
  "before": "0x0",
  "after": "0x3ff",
  "threshold": "0x400"
}` && req.URL.RawQuery == "address=0x0000000000000000000000000000000000000000"+
			"&before=0x0&after=0x3ff&threshold=0x400"
	}))
}

func TestClientWalletReachedFundsThresholdOKNoCalled(t *testing.T) {
	bodyTmplFmt := `{
  "address": "{{.Address}}",
  "before": "{{.Before}}",
  "after": "{{.After}}",
  "threshold": "{{.Threshold}}"
}`
	queryTmplFmt := "address={{.Address}}&before={{.Before}}" +
		"&after={{.After}}&threshold={{.Threshold}}"
	bodyTmpl, err := template.New("WalletReachedFundsThreshold").Parse(bodyTmplFmt)
	assert.Nil(t, err)

	queryURLTmpl, err := template.New("WalletReachedFundsThreshold").Parse(queryTmplFmt)
	assert.Nil(t, err)

	client := newClient()
	mockclient := client.client.(*MockHttpClient)
	client.callbacks = Callbacks{
		WalletReachedFundsThreshold: WalletReachedFundsThresholdCallback{
			Callback: Callback{
				Enabled:        true,
				Method:         http.MethodPost,
				URL:            "http://localhost:1234/",
				BodyFormat:     bodyTmpl,
				QueryURLFormat: queryURLTmpl,
				Headers:        []string{"Content-type:plain/text"},
				Sync:           true,
			},
			Threshold: new(big.Int).SetInt64(0),
		},
	}

	mockclient.On("Do", mock.Anything).
		Return(&http.Response{StatusCode: http.StatusOK}, nil)

	client.WalletReachedFundsThreshold(context.TODO(), WalletReachedFundsThresholdBody{
		Before:  nil,
		After:   new(big.Int).SetInt64(1023),
		Address: "0x0000000000000000000000000000000000000000",
	})

	assert.Nil(t, err)

	mockclient.AssertNotCalled(t, "Do", mock.Anything)
}
