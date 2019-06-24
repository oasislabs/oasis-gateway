package mock

import (
	"context"
	"crypto/ecdsa"
	"fmt"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/oasislabs/developer-gateway/auth"
	"github.com/oasislabs/developer-gateway/backend"
	"github.com/oasislabs/developer-gateway/backend/core"
	"github.com/oasislabs/developer-gateway/backend/eth"
	"github.com/oasislabs/developer-gateway/callback/callbacktest"
	"github.com/oasislabs/developer-gateway/gateway"
	"github.com/oasislabs/developer-gateway/mqueue"
	"github.com/oasislabs/developer-gateway/tx"
)

func NewBackendClient(
	ctx context.Context,
	services *backend.ClientServices,
	config *backend.Config,
) (core.Client, error) {
	switch config.Provider {
	case backend.BackendEthereum:
		return NewEthClient(ctx, services, config.BackendConfig.(*backend.EthereumConfig))
	case backend.BackendEkiden:
		return nil, backend.ErrEkidenBackendNotImplemented
	default:
		return nil, backend.ErrUnknownBackend{Backend: config.Provider.String()}
	}
}

func NewEthClient(
	ctx context.Context,
	services *backend.ClientServices,
	config *backend.EthereumConfig,
) (*eth.Client, error) {
	privateKey, err := crypto.HexToECDSA(config.WalletConfig.PrivateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to read private key with error %s", err.Error())
	}

	mockclient := EthMockClient{}
	executor, err := tx.NewExecutor(ctx, &tx.ExecutorServices{
		Logger:    gateway.RootLogger,
		Client:    mockclient,
		Callbacks: services.Callbacks,
	}, &tx.ExecutorProps{
		PrivateKeys: []*ecdsa.PrivateKey{privateKey},
	})
	if err != nil {
		return nil, err
	}

	client, err := backend.NewEthClientWithDeps(ctx, &eth.ClientDeps{
		Logger:   gateway.RootLogger,
		Client:   mockclient,
		Executor: executor,
	})

	if err != nil {
		return nil, fmt.Errorf("failed to initialize eth client with error %s", err.Error())
	}

	return client, nil
}

func NewServices(ctx context.Context, config *gateway.Config) (*gateway.ServiceGroup, error) {
	mqueue, err := mqueue.NewMailbox(ctx, mqueue.Services{Logger: gateway.RootLogger}, &config.MailboxConfig)
	if err != nil {
		return nil, err
	}

	backendClient, err := NewBackendClient(ctx, &backend.ClientServices{
		Logger:    gateway.RootLogger,
		Callbacks: &callbacktest.MockClient{},
	}, &config.BackendConfig)
	if err != nil {
		return nil, err
	}

	request, err := backend.NewRequestManagerWithDeps(ctx, &backend.Deps{
		Logger: gateway.RootLogger,
		MQueue: mqueue,
		Client: backendClient,
	})
	if err != nil {
		return nil, err
	}

	authenticator, err := auth.NewAuth(&config.AuthConfig)
	if err != nil {
		return nil, err
	}

	return &gateway.ServiceGroup{
		Request:       request,
		Authenticator: authenticator,
	}, nil
}
