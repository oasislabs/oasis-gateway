package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/oasislabs/developer-gateway/api/v0/event"
	"github.com/oasislabs/developer-gateway/api/v0/service"
	"github.com/oasislabs/developer-gateway/auth/core"
	auth "github.com/oasislabs/developer-gateway/auth/core"
	"github.com/oasislabs/developer-gateway/auth/insecure"
	"github.com/oasislabs/developer-gateway/auth/oauth"
	backend "github.com/oasislabs/developer-gateway/backend/core"
	"github.com/oasislabs/developer-gateway/backend/eth"
	"github.com/oasislabs/developer-gateway/log"
	mqueue "github.com/oasislabs/developer-gateway/mqueue/core"
	"github.com/oasislabs/developer-gateway/mqueue/mem"
	"github.com/oasislabs/developer-gateway/mqueue/redis"
	"github.com/oasislabs/developer-gateway/rpc"
	"github.com/sirupsen/logrus"
	"github.com/spf13/pflag"
)

var logger = log.NewLogrus(log.LogrusLoggerProperties{
	Level: logrus.DebugLevel,
})

var rootCtx = context.Background()

type Services struct {
	Request *backend.RequestManager
}

func createServices(ctx context.Context, config ConfigProvider) Services {
	return Services{
		Request: createRequestManager(ctx, config.Get()),
	}
}

func createEthClient(config Config) *eth.EthClient {
	if len(config.Wallet.PrivateKey) == 0 {
		panic("private_key not set in configuration")
	}

	privateKey, err := crypto.HexToECDSA(config.Wallet.PrivateKey)
	if err != nil {
		panic(fmt.Sprintf("failed to read private key with error %s", err.Error()))
	}

	client, err := eth.DialContext(rootCtx, logger, eth.EthClientProperties{
		Wallet: eth.Wallet{
			PrivateKey: privateKey,
		},
		URL: config.EthConfig.URL,
	})

	if err != nil {
		panic(fmt.Sprintf("failed to initialize eth client with error %s", err.Error()))
	}

	return client
}

func createRequestManager(ctx context.Context, config Config) *backend.RequestManager {
	mqueue := createMQueue(ctx, config.MQueueConfig)
	return backend.NewRequestManager(backend.RequestManagerProperties{
		MQueue: mqueue,
		Client: createEthClient(config),
		Logger: logger,
	})
}

func createRedisMQueue(ctx context.Context, config MQueueConfig) mqueue.MQueue {
	if config.Backend != "redis" {
		panic("attempt to create redis backend when it is not in configuration")
	}

	switch config.Mode {
	case "single":
		m, err := redis.NewSingleMQueue(redis.SingleInstanceProps{
			Props: redis.Props{
				Context: ctx,
				Logger:  logger,
			},
			Addr: config.Addr,
		})
		if err != nil {
			panic(fmt.Sprintf("failed to start redis mqueue %s", err.Error()))
		}
		return m
	case "cluster":
		m, err := redis.NewClusterMQueue(redis.ClusterProps{
			Props: redis.Props{
				Context: ctx,
				Logger:  logger,
			},
			Addrs: config.Addrs,
		})
		if err != nil {
			panic(fmt.Sprintf("failed to start redis mqueue %s", err.Error()))
		}
		return m
	default:
		panic(fmt.Sprintf("unknown redis mode %s", config.Mode))
	}
}

func createMQueue(ctx context.Context, config MQueueConfig) mqueue.MQueue {
	switch config.Backend {
	case "redis":
		return createRedisMQueue(ctx, config)
	case "mem":
		return mem.NewServer(ctx, logger)
	default:
		panic(fmt.Sprintf("unknown mqueue backend %s", config.Backend))
	}
}

func createRouter(services Services, verifier auth.Auth) *rpc.HttpRouter {
	binder := rpc.NewHttpBinder(rpc.HttpBinderProperties{
		Encoder: rpc.JsonEncoder{},
		Logger:  logger,
		HandlerFactory: rpc.HttpHandlerFactoryFunc(func(factory rpc.EntityFactory, handler rpc.Handler) rpc.HttpMiddleware {
			jsonHandler := rpc.NewHttpJsonHandler(rpc.HttpJsonHandlerProperties{
				Limit:   1 << 16,
				Handler: handler,
				Logger:  logger,
				Factory: factory,
			})

			return auth.NewHttpMiddlewareAuth(verifier, logger, jsonHandler)
		}),
	})

	service.BindHandler(service.Services{
		Logger:  logger,
		Request: services.Request,
	}, binder)
	event.BindHandler(event.Services{
		Logger:  logger,
		Request: services.Request,
	}, binder)

	return binder.Build()
}

func getAuth(authenticator string) core.Auth {
	switch authenticator {
	case "oauth":
		return oauth.GoogleOauth{}
	case "insecure":
		return insecure.InsecureAuth{}
	default:
		panic("A valid authenticator must be specified")
	}
}

func main() {
	var (
		configFile    string
		authenticator string
	)

	pflag.StringVar(&configFile, "config",
		"cmd/gateway/config/testing.toml",
		"configuration file for the gateway")
	pflag.StringVar(&authenticator, "auth",
		"insecure",
		"which authenticator to use")
	pflag.Parse()

	provider, err := ParseSimpleConfig(configFile)
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}

	ctx := context.Background()

	bindConfig := provider.Get().Bind
	err = bindConfig.Verify(BindConfig{
		HttpInterface:      "127.0.0.1",
		HttpPort:           1234,
		HttpReadTimeoutMs:  10000,
		HttpWriteTimeoutMs: 10000,
		HttpMaxHeaderBytes: 1 << 10,
	})
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}

	httpInterface := bindConfig.HttpInterface
	httpPort := bindConfig.HttpPort

	services := createServices(ctx, provider)
	router := createRouter(services, getAuth(authenticator))

	s := &http.Server{
		Addr:           fmt.Sprintf("%s:%d", httpInterface, httpPort),
		Handler:        router,
		ReadTimeout:    time.Duration(bindConfig.HttpReadTimeoutMs) * time.Millisecond,
		WriteTimeout:   time.Duration(bindConfig.HttpWriteTimeoutMs) * time.Millisecond,
		MaxHeaderBytes: bindConfig.HttpMaxHeaderBytes,
	}

	if err := s.ListenAndServe(); err != nil {
		logger.Fatal(ctx, "http server failed to listen", log.MapFields{
			"err": err.Error(),
		})
	}
}
