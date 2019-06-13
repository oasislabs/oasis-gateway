package mock

import (
	"context"

	"github.com/oasislabs/developer-gateway/auth"
	"github.com/oasislabs/developer-gateway/backend"
	"github.com/oasislabs/developer-gateway/gateway"
	"github.com/oasislabs/developer-gateway/mqueue"
)

func NewServices(ctx context.Context, config *gateway.Config) (gateway.Services, error) {
	mqueue, err := mqueue.NewMailbox(ctx, mqueue.Services{Logger: gateway.RootLogger}, &config.MailboxConfig)
	if err != nil {
		return gateway.Services{}, err
	}

	ethClient, err := backend.NewEthClientWithDeps(ctx, backend.EthClientDeps{
		Logger: gateway.RootLogger,
		Client: EthMockClient{},
	}, config.BackendConfig.BackendConfig.(*backend.EthereumConfig))
	if err != nil {
		return gateway.Services{}, err
	}

	request, err := backend.NewRequestManagerWithDeps(ctx, backend.Deps{
		Logger: gateway.RootLogger,
		MQueue: mqueue,
		Client: ethClient,
	}, &config.BackendConfig)
	if err != nil {
		return gateway.Services{}, err
	}

	authenticator, err := auth.NewAuth(&config.AuthConfig)
	if err != nil {
		return gateway.Services{}, err
	}

	return gateway.Services{
		Request:       request,
		Authenticator: authenticator,
	}, nil
}
