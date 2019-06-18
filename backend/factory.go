package backend

import (
	"context"

	"github.com/oasislabs/developer-gateway/backend/core"
	"github.com/oasislabs/developer-gateway/backend/eth"
	"github.com/oasislabs/developer-gateway/log"
	mqueue "github.com/oasislabs/developer-gateway/mqueue/core"
)

type Deps struct {
	Logger log.Logger
	MQueue mqueue.MQueue
	Client core.Client
}

type EthClientFactory interface {
	New(context.Context, *eth.Deps, *EthereumConfig) (core.Client, error)
}

type EthClientFactoryFunc func(context.Context, *eth.Deps, *EthereumConfig) (core.Client, error)

func (f EthClientFactoryFunc) New(ctx context.Context, deps *eth.Deps, config *EthereumConfig) (core.Client, error) {
	return f(ctx, deps, config)
}

type RequestManagerFactory interface {
	New(ctx context.Context, deps *Deps) (*core.RequestManager, error)
}

type RequestManagerFactoryFunc func(ctx context.Context, deps *Deps) (*core.RequestManager, error)

func (f RequestManagerFactoryFunc) New(ctx context.Context, deps *Deps) (*core.RequestManager, error) {
	return f(ctx, deps)
}

var NewRequestManagerWithDeps = RequestManagerFactoryFunc(func(ctx context.Context, deps *Deps) (*core.RequestManager, error) {
	return core.NewRequestManager(core.RequestManagerProperties{
		MQueue: deps.MQueue,
		Client: deps.Client,
		Logger: deps.Logger,
	}), nil
})

var NewEthClient = EthClientFactoryFunc(func(ctx context.Context, deps *eth.Deps, config *EthereumConfig) (core.Client, error) {
	return eth.NewClient(ctx, deps), nil
})
