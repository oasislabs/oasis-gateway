package eth

import (
	"context"
	"errors"
	"fmt"
	"net/url"

	"github.com/oasislabs/developer-gateway/concurrent"
)

// ClientFactory creates a new instance of a client based
// on the provided configuration
type ClientFactory interface {
	// New creates a new instance of the client
	New(context.Context, *Config) (Client, error)
}

// ClientFactoryFunc allows for functions to act as a ClientFactory
type ClientFactoryFunc func(context.Context, *Config) (Client, error)

// New implementation of ClientFactory for ClientFactoryFunc
func (f ClientFactoryFunc) New(ctx context.Context, config *Config) (Client, error) {
	return f(ctx, config)
}

// NewClient creates a new client with the provided configuration
var NewClient = ClientFactoryFunc(func(ctx context.Context, config *Config) (Client, error) {
	if len(config.URL) == 0 {
		return nil, errors.New("no url provided for eth client")
	}

	url, err := url.Parse(config.URL)
	if err != nil {
		return nil, fmt.Errorf("Failed to parse url %s", err.Error())
	}

	if url.Scheme != "wss" && url.Scheme != "ws" {
		return nil, errors.New("Only schemes supported are ws and wss")
	}

	dialer := NewUniDialer(ctx, config.URL)
	return NewPooledClient(PooledClientProps{
		Pool:        dialer,
		RetryConfig: concurrent.RandomConfig,
	}), nil
})
