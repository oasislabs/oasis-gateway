package backend

import (
	"context"
	"crypto/ecdsa"
	"fmt"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/oasislabs/developer-gateway/backend/core"
	"github.com/oasislabs/developer-gateway/backend/eth"
	ethereum "github.com/oasislabs/developer-gateway/eth"
	"github.com/oasislabs/developer-gateway/log"
	mqueue "github.com/oasislabs/developer-gateway/mqueue/core"
)

type Deps struct {
	Logger log.Logger
	MQueue mqueue.MQueue
	Client core.Client
}

type Services struct {
	Logger log.Logger
	MQueue mqueue.MQueue
}

func NewRequestManagerWithDeps(ctx context.Context, deps Deps, config *Config) (*core.RequestManager, error) {
	return core.NewRequestManager(core.RequestManagerProperties{
		MQueue: deps.MQueue,
		Client: deps.Client,
		Logger: deps.Logger,
	}), nil
}

func NewRequestManager(ctx context.Context, services Services, config *Config) (*core.RequestManager, error) {
	client, err := NewBackendClient(ctx, Services{Logger: services.Logger}, config)
	if err != nil {
		return nil, err
	}

	return NewRequestManagerWithDeps(ctx, Deps{
		MQueue: services.MQueue,
		Logger: services.Logger,
		Client: client,
	}, config)
}

func NewBackendClient(ctx context.Context, services Services, config *Config) (core.Client, error) {
	switch config.Provider {
	case BackendEthereum:
		return NewEthClient(ctx, EthClientServices{
			Logger: services.Logger,
		}, config.BackendConfig.(*EthereumConfig))
	case BackendEkiden:
		return nil, ErrEkidenBackendNotImplemented
	default:
		return nil, ErrUnknownBackend{Backend: config.Provider.String()}
	}
}

type EthClientFactoryFunc func(context.Context, *EthereumConfig) (*eth.EthClient, error)

type EthClientDeps struct {
	Logger log.Logger
	Client ethereum.Client
}

type EthClientServices struct {
	Logger log.Logger
}

func NewEthClientWithDeps(ctx context.Context, deps EthClientDeps, config *EthereumConfig) (*eth.EthClient, error) {
	privateKey, err := crypto.HexToECDSA(config.WalletConfig.PrivateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to read private key with error %s", err.Error())
	}

	return eth.NewClient(ctx, deps.Logger, []*ecdsa.PrivateKey{privateKey}, deps.Client)
}

func NewEthClient(ctx context.Context, services EthClientServices, config *EthereumConfig) (*eth.EthClient, error) {
	privateKey, err := crypto.HexToECDSA(config.WalletConfig.PrivateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to read private key with error %s", err.Error())
	}

	client, err := eth.DialContext(ctx, services.Logger, eth.EthClientProperties{
		PrivateKeys: []*ecdsa.PrivateKey{privateKey},
		URL:         config.URL,
	})

	if err != nil {
		return nil, fmt.Errorf("failed to initialize eth client with error %s", err.Error())
	}

	return client, nil
}
