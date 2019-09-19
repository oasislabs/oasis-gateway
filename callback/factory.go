package callback

import (
	"context"
	"html/template"
	"math/big"
	"net/http"
	"time"

	"github.com/oasislabs/developer-gateway/callback/client"
	"github.com/oasislabs/developer-gateway/log"
)

type ClientDeps struct {
	Logger log.Logger
	Client client.HttpClient
}

type ClientServices struct {
	Logger log.Logger
}

type ClientFactory interface {
	New(ctx context.Context, services *ClientServices, config *Config) (*client.Client, error)
}

type CallbacksFactoryFunc func(ctx context.Context, services *ClientServices, config *Config) (*client.Client, error)

func (f CallbacksFactoryFunc) New(ctx context.Context, services *ClientServices, config *Config) (*client.Client, error) {
	return f(ctx, services, config)
}

func parseCallback(name string, callback Callback) (client.Callback, error) {
	var (
		bodyFormat     *template.Template
		queryURLFormat *template.Template
	)
	if len(callback.Body) > 0 {
		tmpl, err := template.New(name).Parse(callback.Body)
		if err != nil {
			return client.Callback{}, err
		}

		bodyFormat = tmpl
	}

	if len(callback.QueryURL) > 0 {
		tmpl, err := template.New(name).Parse(callback.QueryURL)
		if err != nil {
			return client.Callback{}, err
		}

		queryURLFormat = tmpl
	}

	return client.Callback{
		Enabled:        callback.Enabled,
		Name:           name,
		Method:         callback.Method,
		URL:            callback.URL,
		BodyFormat:     bodyFormat,
		QueryURLFormat: queryURLFormat,
		Headers:        callback.Headers,
		Sync:           callback.Sync,
		PeriodLimit:    1 * time.Minute,
	}, nil
}

func parseWalletReachedFundsThresholdCallback(config WalletReachedFundsThreshold) (client.WalletReachedFundsThresholdCallback, error) {
	callback, err := parseCallback("WalletReachedFundsThreshold", Callback{
		Enabled:  config.Enabled,
		Sync:     config.Sync,
		Method:   config.Method,
		URL:      config.URL,
		Body:     config.Body,
		QueryURL: config.QueryURL,
		Headers:  config.Headers,
	})
	if err != nil {
		return client.WalletReachedFundsThresholdCallback{}, err
	}

	return client.WalletReachedFundsThresholdCallback{
		Callback:  callback,
		Threshold: new(big.Int).SetUint64(config.Threshold),
	}, nil
}

func NewClientWithDeps(ctx context.Context, deps *client.Deps, config *Config) (*client.Client, error) {
	transactionCommitted, err := parseCallback("TransactionCommitted", config.TransactionCommitted.Callback)
	if err != nil {
		return nil, err
	}

	walletOutOfFunds, err := parseCallback("WalletOutOfFundsBody", config.WalletOutOfFunds.Callback)
	if err != nil {
		return nil, err
	}

	walletReachedFundsThreshold, err := parseWalletReachedFundsThresholdCallback(config.WalletReachedFundsThreshold)
	if err != nil {
		return nil, err
	}

	return client.NewClientWithDeps(deps, &client.Props{
		Callbacks: client.Callbacks{
			TransactionCommitted:        transactionCommitted,
			WalletOutOfFunds:            walletOutOfFunds,
			WalletReachedFundsThreshold: walletReachedFundsThreshold,
		},
	}), nil
}

// NewClient creates a new instance of the client with the
// specified configuration and the provided services
var NewClient = CallbacksFactoryFunc(func(ctx context.Context, services *ClientServices, config *Config) (*client.Client, error) {
	return NewClientWithDeps(ctx, &client.Deps{
		Logger: services.Logger,
		Client: &http.Client{},
	}, config)
})
