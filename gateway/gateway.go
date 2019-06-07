package gateway

import (
	"context"
	"crypto/ecdsa"
	"errors"
	"fmt"
	"net/url"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/oasislabs/developer-gateway/api/v0/event"
	"github.com/oasislabs/developer-gateway/api/v0/health"
	"github.com/oasislabs/developer-gateway/api/v0/service"
	"github.com/oasislabs/developer-gateway/auth/core"
	auth "github.com/oasislabs/developer-gateway/auth/core"
	"github.com/oasislabs/developer-gateway/auth/insecure"
	"github.com/oasislabs/developer-gateway/auth/oauth"
	backend "github.com/oasislabs/developer-gateway/backend/core"
	"github.com/oasislabs/developer-gateway/backend/eth"
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

type EthClientFactoryFunc func(context.Context, *config.Config) (*eth.EthClient, error)

func NewRedisMQueue(ctx context.Context, conf config.MailboxConfig) (mqueue.MQueue, error) {
	if conf.Mailbox.ID() != config.MailboxRedisSingle &&
		conf.Mailbox.ID() != config.MailboxRedisCluster {
		return nil, errors.New("attempt to create redis backend when it is not in confuration")
	}

	switch conf.Mailbox.ID() {
	case config.MailboxRedisSingle:
		conf := conf.Mailbox.(*config.MailboxRedisSingleConfig)
		m, err := redis.NewSingleMQueue(redis.SingleInstanceProps{
			Props: redis.Props{
				Context: ctx,
				Logger:  RootLogger,
			},
			Addr: conf.Addr,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to start redis mqueue %s", err.Error())
		}
		return m, nil
	case config.MailboxRedisCluster:
		conf := conf.Mailbox.(*config.MailboxRedisClusterConfig)
		m, err := redis.NewClusterMQueue(redis.ClusterProps{
			Props: redis.Props{
				Context: ctx,
				Logger:  RootLogger,
			},
			Addrs: conf.Addrs,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to start redis mqueue %s", err.Error())
		}
		return m, nil
	default:
		panic("cannot reach")
	}
}

func NewMQueue(ctx context.Context, conf config.MailboxConfig) (mqueue.MQueue, error) {
	switch conf.Mailbox.ID() {
	case config.MailboxRedisSingle:
		return NewRedisMQueue(ctx, conf)
	case config.MailboxRedisCluster:
		return NewRedisMQueue(ctx, conf)
	case config.MailboxMem:
		return mem.NewServer(ctx, RootLogger), nil
	default:
		panic(fmt.Sprintf("unknown backend %s", conf.Mailbox.ID()))
	}
}

func NewEthClient(ctx context.Context, conf *config.Config) (*eth.EthClient, error) {
	if len(conf.WalletConfig.PrivateKeys) == 0 {
		return nil, errors.New("private_keys not set in configuration")
	}

	privateKeys := make([]*ecdsa.PrivateKey, len(conf.WalletConfig.PrivateKeys))
	for i := 0; i < len(conf.WalletConfig.PrivateKeys); i++ {
		privateKey, err := crypto.HexToECDSA(conf.WalletConfig.PrivateKeys[i])
		if err != nil {
			return nil, fmt.Errorf("failed to read private key with error %s", err.Error())
		}
		privateKeys[i] = privateKey
	}

	if len(conf.EthConfig.URL) == 0 {
		return nil, fmt.Errorf("no url provided for eth client")
	}

	url, err := url.Parse(conf.EthConfig.URL)
	if err != nil {
		return nil, fmt.Errorf("Failed to parse url %s", err.Error())
	}

	if url.Scheme != "wss" && url.Scheme != "ws" {
		return nil, fmt.Errorf("Only schemes supported are ws and wss")
	}

	client, err := eth.DialContext(ctx, RootLogger, eth.EthClientProperties{
		PrivateKeys: privateKeys,
		URL:         conf.EthConfig.URL,
	})

	if err != nil {
		return nil, fmt.Errorf("failed to initialize eth client with error %s", err.Error())
	}

	return client, nil
}

func NewRequestManager(ctx context.Context, mqueue mqueue.MQueue, client backend.Client, config *config.Config) (*backend.RequestManager, error) {
	return backend.NewRequestManager(backend.RequestManagerProperties{
		MQueue: mqueue,
		Client: client,
		Logger: RootLogger,
	}), nil
}

func NewServices(ctx context.Context, conf *config.Config, factories Factories) (Services, error) {
	mqueue, err := NewMQueue(ctx, conf.MailboxConfig)
	if err != nil {
		return Services{}, err
	}

	client, err := factories.EthClientFactory(ctx, conf)
	if err != nil {
		return Services{}, err
	}

	request, err := NewRequestManager(ctx, mqueue, client, conf)
	if err != nil {
		return Services{}, err
	}

	authenticator, err := NewAuth(conf.AuthConfig.Provider)
	if err != nil {
		return Services{}, err
	}

	return Services{
		Request:       request,
		Authenticator: authenticator,
	}, nil
}

func NewPrivateRouter() *rpc.HttpRouter {
	binder := rpc.NewHttpBinder(rpc.HttpBinderProperties{
		Encoder: rpc.JsonEncoder{},
		Logger:  RootLogger,
		HandlerFactory: rpc.HttpHandlerFactoryFunc(func(factory rpc.EntityFactory, handler rpc.Handler) rpc.HttpMiddleware {
			// TODO(stan): we may want to add some authentication mechanism
			// to the private router
			return rpc.NewHttpJsonHandler(rpc.HttpJsonHandlerProperties{
				Limit:   1 << 16,
				Handler: handler,
				Logger:  RootLogger,
				Factory: factory,
			})
		}),
	})

	health.BindHandler(health.Services{}, binder)

	return binder.Build()
}

func NewPublicRouter(services Services) *rpc.HttpRouter {
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
