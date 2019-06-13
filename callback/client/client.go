package client

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/oasislabs/developer-gateway/conc"
	"github.com/oasislabs/developer-gateway/log"
)

// Calls are all the callbacks that the client implements
type Calls interface {
	WalletOutOfFunds(ctx context.Context, body WalletOutOfFundsBody) error
}

// HttpClient is the basic interface for the
// underlying http client used by the Client
type HttpClient interface {
	Do(*http.Request) (*http.Response, error)
}

// Callbacks defines all the callbacks that the
// client supports and the behaviour that the client
// should have on those callbacks
type Callbacks struct {
	WalletOutOfFunds Callback
}

// Services are services required by the client
type Services struct {
	Logger log.Logger
}

// Props are the properties that define
// the behaviour of the client to send callbacks
type Props struct {
	Callbacks   Callbacks
	RetryConfig conc.RetryConfig
}

// Deps are the required instantiated dependencies
// that a Client requires
type Deps struct {
	Logger log.Logger
	Client HttpClient
}

// NewClient creates a new callback client
func NewClient(services *Services, props *Props) *Client {
	return NewClientWithDeps(&Deps{
		Logger: services.Logger,
		Client: &http.Client{},
	}, props)
}

// NewClientWithDeps creates a new client using the external
// dependencies provided
func NewClientWithDeps(deps *Deps, props *Props) *Client {
	return &Client{
		callbacks:   props.Callbacks,
		retryConfig: props.RetryConfig,
		client:      deps.Client,
	}
}

// Client is the callback client that will send
// callbacks when events are triggered
type Client struct {
	callbacks   Callbacks
	client      HttpClient
	retryConfig conc.RetryConfig
	logger      log.Logger
}

// request sends an http request
func (c *Client) request(ctx context.Context, req *http.Request) error {
	_, err := conc.RetryWithConfig(ctx, conc.SupplierFunc(func() (interface{}, error) {
		res, err := c.client.Do(req)
		if err != nil {
			return nil, err
		}

		if res.StatusCode >= 500 {
			return nil, ErrDeliverHttpRequest{
				Cause: fmt.Errorf("http request failed with status %d", res.StatusCode),
			}
		}

		return nil, nil
	}), c.retryConfig)
	if err != nil {
		return err
	}

	return nil
}

func (c *Client) callback(ctx context.Context, callback *Callback) error {
	if !callback.Enabled {
		return nil
	}

	req, err := http.NewRequest(callback.Method, callback.URL, nil)
	if err != nil {
		return ErrNewHttpRequest{Cause: err}
	}

	for _, header := range callback.Headers {
		h := strings.SplitN(header, ":", 2)
		if len(h) != 2 {
			continue
		}

		req.Header.Add(h[0], h[1])
	}

	return c.request(ctx, req)
}

// WalletOutOfFunds sends a callback that is triggered when a wallet
// is out of funds
func (c *Client) WalletOutOfFunds(ctx context.Context, body WalletOutOfFundsBody) error {
	c.logger.Debug(ctx, "", log.MapFields{
		"call_type": "SendWalletOutOfFundsAttempt",
		"address":   body.Address,
	})

	err := c.callback(ctx, &c.callbacks.WalletOutOfFunds)
	if err != nil {
		c.logger.Warn(ctx, "", log.MapFields{
			"call_type": "SendWalletOutOfFundsFailure",
			"address":   body.Address,
			"err":       err.Error(),
		})
		return err
	}

	c.logger.Info(ctx, "", log.MapFields{
		"call_type": "SendWalletOutOfFundsSuccess",
		"address":   body.Address,
	})
	return nil
}
