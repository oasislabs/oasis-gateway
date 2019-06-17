package callback

import (
	"context"
	"html/template"
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

// NewClient creates a new instance of the client with the
// specified configuration and the provided services
var NewClient = CallbacksFactoryFunc(func(ctx context.Context, services *ClientServices, config *Config) (*client.Client, error) {
	var bodyFormat *template.Template
	if len(config.WalletOutOfFunds.Body) > 0 {
		tmpl, err := template.New("WalletOutOfFunds").Parse(config.WalletOutOfFunds.Body)
		if err != nil {
			return nil, err
		}

		bodyFormat = tmpl
	}

	return client.NewClient(&client.Services{
		Logger: services.Logger,
	}, &client.Props{
		Callbacks: client.Callbacks{
			WalletOutOfFunds: client.Callback{
				Enabled:     config.WalletOutOfFunds.Enabled,
				Name:        "WalletOutOfFunds",
				Method:      config.WalletOutOfFunds.Method,
				URL:         config.WalletOutOfFunds.URL,
				BodyFormat:  bodyFormat,
				Headers:     config.WalletOutOfFunds.Headers,
				PeriodLimit: 1 * time.Minute,
			},
		},
	}), nil
})
