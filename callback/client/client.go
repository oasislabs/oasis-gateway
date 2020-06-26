package client

import (
	"bytes"
	"context"
	"fmt"
	"math/big"
	"net/http"
	"strings"
	"time"

	"github.com/oasislabs/oasis-gateway/metrics"

	"github.com/oasislabs/oasis-gateway/concurrent"
	"github.com/oasislabs/oasis-gateway/log"
	"github.com/oasislabs/oasis-gateway/stats"
)

const walletOutOfFunds string = "WalletOutOfFunds"

// CallbackProps are properties that can be passed
// when executing a callback to modify the behaviour
// of the call
type CallbackProps struct {
	// Body is the type that will be used by for the
	// template to generate the body that will be
	// sent on the request
	Body interface{}
}

// Calls are all the callbacks that the client implements
type Calls interface {
	TransactionCommitted(ctx context.Context, body TransactionCommittedBody)
	WalletOutOfFunds(ctx context.Context, body WalletOutOfFundsBody)
	WalletReachedFundsThreshold(ctx context.Context, body WalletReachedFundsThresholdBody)
}

// HttpClient is the basic interface for the
// underlying http client used by the Client
type HttpClient interface {
	Do(*http.Request) (*http.Response, error)
}

type WalletReachedFundsThresholdCallback struct {
	Callback
	Threshold *big.Int
}

// Callbacks defines all the callbacks that the
// client supports and the behaviour that the client
// should have on those callbacks
type Callbacks struct {
	TransactionCommitted        Callback
	WalletOutOfFunds            Callback
	WalletReachedFundsThreshold WalletReachedFundsThresholdCallback
}

// Services are services required by the client
type Services struct {
	Logger log.Logger
}

// Props are the properties that define
// the behaviour of the client to send callbacks
type Props struct {
	Callbacks   Callbacks
	RetryConfig concurrent.RetryConfig
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
		logger:      deps.Logger,
		tracker:     stats.NewMethodTracker(walletOutOfFunds),
		metrics:     metrics.NewDefaultServiceMetrics("oasis-gateway-callback"),
	}
}

// Client is the callback client that will send
// callbacks when events are triggered
type Client struct {
	callbacks   Callbacks
	client      HttpClient
	retryConfig concurrent.RetryConfig
	logger      log.Logger
	tracker     *stats.MethodTracker
	metrics     *metrics.ServiceMetrics
}

func (c *Client) Stats() stats.Metrics {
	return c.tracker.Stats()
}

func (c *Client) Name() string {
	return "callback.client.Client"
}

func (c *Client) instrumentedRequest(ctx context.Context, method string, req *http.Request) (int, error) {
	timer := c.metrics.RequestTimer(method)
	defer timer.ObserveDuration()

	code, err := c.request(ctx, req)
	if err != nil {
		c.metrics.RequestCounter(method, "fail").Inc()
		return 0, err
	}

	c.metrics.RequestCounter(method, "success").Inc()
	return code, err
}

// request sends an http request
func (c *Client) request(ctx context.Context, req *http.Request) (int, error) {
	code, err := concurrent.RetryWithConfig(ctx, concurrent.SupplierFunc(func() (interface{}, error) {
		res, err := c.client.Do(req)
		if err != nil {
			return 0, err
		}

		if res.StatusCode >= 500 {
			return 0, fmt.Errorf("http request failed with status %d", res.StatusCode)
		}

		return res.StatusCode, nil
	}), c.retryConfig)

	if err != nil {
		return 0, err
	}

	return code.(int), err
}

func (c *Client) createRequest(
	ctx context.Context,
	callback *Callback,
	props *CallbackProps,
) (*http.Request, error) {
	url := callback.URL

	if callback.QueryURLFormat != nil && props.Body != nil {
		buffer := bytes.NewBuffer([]byte{})
		if err := callback.QueryURLFormat.Execute(buffer, props.Body); err != nil {
			return nil, err
		}

		queryURL := buffer.String()
		if !strings.HasPrefix(queryURL, "?") {
			queryURL = "?" + queryURL
		}
		url += queryURL
	}

	if callback.BodyFormat != nil && props.Body != nil {
		buffer := bytes.NewBuffer([]byte{})
		if err := callback.BodyFormat.Execute(buffer, props.Body); err != nil {
			return nil, err
		}

		s := strings.Replace(buffer.String(), "'", "\"", -1)
		buffer.Reset()
		buffer.WriteString(s)
		return http.NewRequest(callback.Method, url, buffer)
	}

	return http.NewRequest(callback.Method, url, nil)
}

func (c *Client) Callback(
	ctx context.Context,
	callback *Callback,
	props *CallbackProps,
) error {
	if !callback.Enabled {
		return nil
	}

	now := time.Now().Unix()
	if now-callback.LastAttempt < int64(callback.PeriodLimit.Seconds()) {
		return nil
	}

	req, err := c.createRequest(ctx, callback, props)
	if err != nil {
		c.logger.Warn(ctx, "failed to create http request", log.MapFields{
			"call_type": "SendCallbackFailure",
			"method":    callback.Method,
			"url":       callback.URL,
			"err":       err.Error(),
		})
		return err
	}

	for _, header := range callback.Headers {
		h := strings.SplitN(header, ":", 2)
		if len(h) != 2 {
			continue
		}

		req.Header.Add(h[0], h[1])
	}

	c.logger.Debug(ctx, "attempt to deliver http callback", log.MapFields{
		"call_type": "SendCallbackAttempt",
		"method":    callback.Method,
		"url":       callback.URL,
		"callback":  callback.Name,
		"sync":      callback.Sync,
	})

	if callback.Sync {
		return c.deliver(ctx, callback, req)
	}

	go func() {
		_ = c.deliver(ctx, callback, req)
	}()

	return nil
}

func (c *Client) deliver(ctx context.Context, callback *Callback, req *http.Request) error {
	code, err := c.instrumentedRequest(ctx, callback.Method, req)
	if err != nil {
		c.logger.Warn(ctx, "failed to deliver http callback", log.MapFields{
			"call_type":  "SendCallbackFailure",
			"method":     callback.Method,
			"url":        callback.URL,
			"callback":   callback.Name,
			"sync":       callback.Sync,
			"statusCode": code,
			"err":        err.Error(),
		})
	}

	c.logger.Debug(ctx, "http callback delivered", log.MapFields{
		"call_type":  "SendCallbackSuccess",
		"method":     callback.Method,
		"url":        callback.URL,
		"callback":   callback.Name,
		"statusCode": code,
		"sync":       callback.Sync,
	})

	return err
}

// WalletOutOfFunds sends a callback that is triggered when a wallet
// is out of funds
func (c *Client) WalletOutOfFunds(ctx context.Context, body WalletOutOfFundsBody) {
	_ = c.Callback(ctx, &c.callbacks.WalletOutOfFunds, &CallbackProps{
		Body: body,
	})
}

// WalletReachedFundsThreshold sends a callback that is triggered when a wallet
// reaches a specific threshold of funds
func (c *Client) WalletReachedFundsThreshold(ctx context.Context, body WalletReachedFundsThresholdBody) {
	isSet := c.callbacks.WalletReachedFundsThreshold.Threshold != nil &&
		c.callbacks.WalletReachedFundsThreshold.Threshold.Cmp(new(big.Int).SetInt64(0)) > 0

	if isSet &&
		(body.Before == nil ||
			c.callbacks.WalletReachedFundsThreshold.Threshold.Cmp(body.Before) < 0) &&
		body.After != nil &&
		c.callbacks.WalletReachedFundsThreshold.Threshold.Cmp(body.After) > 0 {
		if body.Before == nil {
			body.Before = new(big.Int).SetInt64(0)
		}

		_ = c.Callback(ctx, &c.callbacks.WalletReachedFundsThreshold.Callback, &CallbackProps{
			Body: WalletReachedFundsThresholdRequest{
				Address:   body.Address,
				Before:    fmt.Sprintf("0x%x", body.Before),
				After:     fmt.Sprintf("0x%x", body.After),
				Threshold: fmt.Sprintf("0x%x", c.callbacks.WalletReachedFundsThreshold.Threshold),
			},
		})
	}
}

// TransactionCommitted sends a callback that is triggered when a
// transaction has been committed to the blockchain
func (c *Client) TransactionCommitted(ctx context.Context, body TransactionCommittedBody) {
	_ = c.Callback(ctx, &c.callbacks.TransactionCommitted, &CallbackProps{
		Body: body,
	})
}
