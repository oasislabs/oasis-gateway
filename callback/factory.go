package callback

import (
	"context"
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

// NewClient creates a new instance of the client with the
// specified configuration and the provided services
func NewClient(ctx context.Context, services *ClientServices, config *Config) *client.Client {
	return client.NewClient(&client.Services{
		Logger: services.Logger,
	}, &client.Props{
		Callbacks: client.Callbacks{
			WalletOutOfFunds: client.Callback{
				Enabled:     config.WalletOutOfFunds.Enabled,
				Name:        "WalletOutOfFunds",
				Method:      config.WalletOutOfFunds.Method,
				URL:         config.WalletOutOfFunds.URL,
				Body:        config.WalletOutOfFunds.Body,
				Headers:     config.WalletOutOfFunds.Headers,
				PeriodLimit: 1 * time.Minute,
			},
		},
	})
}
