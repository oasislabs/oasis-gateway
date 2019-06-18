package callback

import (
	"context"
	"html/template"
	"net/http"
	"time"

	"github.com/oasislabs/developer-gateway/callback/client"
	"github.com/oasislabs/developer-gateway/log"
	"github.com/oasislabs/developer-gateway/rpc"
)

type ClientDeps struct {
	Logger log.Logger
	Client rpc.HttpClient
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

func NewClientWithDeps(ctx context.Context, deps *client.Deps, config *Config) (*client.Client, error) {
	var (
		bodyFormat     *template.Template
		queryURLFormat *template.Template
	)
	if len(config.WalletOutOfFunds.Body) > 0 {
		tmpl, err := template.New("WalletOutOfFundsBody").Parse(config.WalletOutOfFunds.Body)
		if err != nil {
			return nil, err
		}

		bodyFormat = tmpl
	}

	if len(config.WalletOutOfFunds.QueryURL) > 0 {
		tmpl, err := template.New("WalletOutOfFundsQueryURL").Parse(config.WalletOutOfFunds.QueryURL)
		if err != nil {
			return nil, err
		}

		queryURLFormat = tmpl
	}

	return client.NewClientWithDeps(deps, &client.Props{
		Callbacks: client.Callbacks{
			WalletOutOfFunds: client.Callback{
				Enabled:        config.WalletOutOfFunds.Enabled,
				Name:           "WalletOutOfFunds",
				Method:         config.WalletOutOfFunds.Method,
				URL:            config.WalletOutOfFunds.URL,
				BodyFormat:     bodyFormat,
				QueryURLFormat: queryURLFormat,
				Headers:        config.WalletOutOfFunds.Headers,
				Sync:           config.WalletOutOfFunds.Sync,
				PeriodLimit:    1 * time.Minute,
			},
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
