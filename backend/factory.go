package backend

import (
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/oasislabs/developer-gateway/backend/core"
	"github.com/oasislabs/developer-gateway/backend/eth"
	callback "github.com/oasislabs/developer-gateway/callback/client"
	"github.com/oasislabs/developer-gateway/log"
	mqueue "github.com/oasislabs/developer-gateway/mqueue/core"
)

type Deps struct {
	Logger log.Logger
	MQueue mqueue.MQueue
	Client core.Client
}

type ClientServices struct {
	Logger    log.Logger
	Callbacks callback.Calls
}

func NewRequestManagerWithDeps(ctx context.Context, deps *Deps) (*core.RequestManager, error) {
	return core.NewRequestManager(core.RequestManagerProperties{
		MQueue: deps.MQueue,
		Client: deps.Client,
		Logger: deps.Logger,
	}), nil
}

func NewBackendClient(ctx context.Context, services *ClientServices, config *Config) (core.Client, error) {
	switch config.Provider {
	case BackendEthereum:
		return NewEthClient(ctx, &eth.ClientServices{
			Logger:    services.Logger,
			Callbacks: services.Callbacks,
		}, config.BackendConfig.(*EthereumConfig))
	case BackendEkiden:
		return nil, ErrEkidenBackendNotImplemented
	default:
		return nil, ErrUnknownBackend{Backend: config.Provider.String()}
	}
}

func NewEthClientWithDeps(ctx context.Context, deps *eth.ClientDeps) (*eth.Client, error) {
	return eth.NewClientWithDeps(ctx, deps), nil
}

func NewEthClient(ctx context.Context, services *eth.ClientServices, config *EthereumConfig) (*eth.Client, error) {
	privateKey, err := crypto.HexToECDSA(config.WalletConfig.PrivateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to read private key with error %s", err.Error())
	}

	client, err := eth.DialContext(ctx, services, &eth.ClientProps{
		PrivateKey: privateKey,
		URL:        config.URL,
	})

	if err != nil {
		return nil, fmt.Errorf("failed to initialize eth client with error %s", err.Error())
	}

	return client, nil
}
