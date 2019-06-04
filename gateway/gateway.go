package gateway

import (
	"context"
	"crypto/ecdsa"
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/oasislabs/developer-gateway/api/v0/event"
	"github.com/oasislabs/developer-gateway/api/v0/service"
	"github.com/oasislabs/developer-gateway/auth/core"
	auth "github.com/oasislabs/developer-gateway/auth/core"
	"github.com/oasislabs/developer-gateway/auth/insecure"
	"github.com/oasislabs/developer-gateway/auth/oauth"
	backend "github.com/oasislabs/developer-gateway/backend/core"
	ethereum "github.com/oasislabs/developer-gateway/backend/eth"
	"github.com/oasislabs/developer-gateway/eth"
	"github.com/oasislabs/developer-gateway/gateway/config"
	"github.com/oasislabs/developer-gateway/log"
	mqueue "github.com/oasislabs/developer-gateway/mqueue/core"
	"github.com/oasislabs/developer-gateway/mqueue/mem"
	"github.com/oasislabs/developer-gateway/mqueue/redis"
	"github.com/oasislabs/developer-gateway/rpc"
	"github.com/sirupsen/logrus"
)

var RootLogger = log.NewLogrus(log.LogrusLoggerProperties{
	Level: logrus.DebugLevel,
})

var RootContext = context.Background()

type Services struct {
	Request       *backend.RequestManager
	Authenticator auth.Auth
}

type Factories struct {
	EthClientFactory EthClientFactoryFunc
}

type EthClientFactoryFunc func(ctx context.Context, config config.Config) (*eth.EthClient, error)

func NewRedisMQueue(ctx context.Context, config config.MQueueConfig) (mqueue.MQueue, error) {
	if config.Backend != "redis" {
		return nil, errors.New("attempt to create redis backend when it is not in configuration")
	}

	switch config.Mode {
	case "single":
		m, err := redis.NewSingleMQueue(redis.SingleInstanceProps{
			Props: redis.Props{
				Context: ctx,
				Logger:  RootLogger,
			},
			Addr: config.Addr,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to start redis mqueue %s", err.Error())
		}
		return m, nil
	case "cluster":
		m, err := redis.NewClusterMQueue(redis.ClusterProps{
			Props: redis.Props{
				Context: ctx,
				Logger:  RootLogger,
			},
			Addrs: config.Addrs,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to start redis mqueue %s", err.Error())
		}
		return m, nil
	default:
		return nil, fmt.Errorf("unknown redis mode %s", config.Mode)
	}
}

func NewMQueue(ctx context.Context, config config.MQueueConfig) (mqueue.MQueue, error) {
	switch config.Backend {
	case "redis":
		return NewRedisMQueue(ctx, config)
	case "mem":
		return mem.NewServer(ctx, RootLogger), nil
	default:
		panic(fmt.Sprintf("unknown mqueue backend %s", config.Backend))
	}
}

func NewEthClient(ctx context.Context, config config.Config) (*eth.EthClient, error) {
	if len(config.Wallet.PrivateKeys) == 0 {
		return nil, errors.New("private_keys not set in configuration")
	}

	privateKeys := make([]*ecdsa.PrivateKey, len(config.Wallet.PrivateKeys))
	for i := 0; i < len(config.Wallet.PrivateKeys); i++ {
		privateKey, err := crypto.HexToECDSA(config.Wallet.PrivateKeys[i])
		if err != nil {
			return nil, fmt.Errorf("failed to read private key with error %s", err.Error())
		}
		privateKeys[i] = privateKey
	}

	if len(config.EthConfig.URL) == 0 {
		return nil, fmt.Errorf("no url provided for eth client")
	}

	url, err := url.Parse(config.EthConfig.URL)
	if err != nil {
		return nil, fmt.Errorf("Failed to parse url %s", err.Error())
	}

	if url.Scheme != "wss" && url.Scheme != "ws" {
		return nil, fmt.Errorf("Only schemes supported are ws and wss")
	}

	client, err := eth.DialContext(ctx, RootLogger, eth.EthClientProperties{
		PrivateKeys: privateKeys,
		URL:         config.EthConfig.URL,
	})

	if err != nil {
		return nil, fmt.Errorf("failed to initialize eth client with error %s", err.Error())
	}

	return client, nil
}

func NewRequestManager(ctx context.Context, mqueue mqueue.MQueue, client backend.Client, config config.Config) (*backend.RequestManager, error) {
	return backend.NewRequestManager(backend.RequestManagerProperties{
		MQueue: mqueue,
		Client: client,
		Logger: RootLogger,
	}), nil
}

func NewServices(ctx context.Context, config config.Config, factories Factories) (Services, error) {
	mqueue, err := NewMQueue(ctx, config.MQueueConfig)
	if err != nil {
		return Services{}, err
	}

	client, err := factories.EthClientFactory(ctx, config)
	if err != nil {
		return Services{}, err
	}

	request, err := NewRequestManager(ctx, mqueue, client, config)
	if err != nil {
		return Services{}, err
	}

	authenticator, err := NewAuth(config.AuthConfig.Provider)
	if err != nil {
		return Services{}, err
	}

	return Services{
		Request:       request,
		Authenticator: authenticator,
	}, nil
}

func NewRouter(services Services) *rpc.HttpRouter {
	binder := rpc.NewHttpBinder(rpc.HttpBinderProperties{
		Encoder: rpc.JsonEncoder{},
		Logger:  RootLogger,
		HandlerFactory: rpc.HttpHandlerFactoryFunc(func(factory rpc.EntityFactory, handler rpc.Handler) rpc.HttpMiddleware {
			jsonHandler := rpc.NewHttpJsonHandler(rpc.HttpJsonHandlerProperties{
				Limit:   1 << 16,
				Handler: handler,
				Logger:  RootLogger,
				Factory: factory,
			})

			return auth.NewHttpMiddlewareAuth(services.Authenticator, RootLogger, jsonHandler)
		}),
	})

	service.BindHandler(service.Services{
		Logger:   RootLogger,
		Client:   services.Request,
		Verifier: auth.TrustedPayloadVerifier{},
	}, binder)
	event.BindHandler(event.Services{
		Logger:  RootLogger,
		Request: services.Request,
	}, binder)

	return binder.Build()
}

func NewAuth(provider string) (core.Auth, error) {
	switch provider {
	case "oauth":
		return oauth.GoogleOauth{}, nil
	case "insecure":
		return insecure.InsecureAuth{}, nil
	default:
		return nil, errors.New("A valid authenticator must be specified")
	}
}
