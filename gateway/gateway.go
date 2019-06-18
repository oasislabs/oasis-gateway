package gateway

import (
	"context"

	"github.com/oasislabs/developer-gateway/api/v0/event"
	"github.com/oasislabs/developer-gateway/api/v0/health"
	"github.com/oasislabs/developer-gateway/api/v0/service"
	"github.com/oasislabs/developer-gateway/auth"
	authcore "github.com/oasislabs/developer-gateway/auth/core"
	"github.com/oasislabs/developer-gateway/backend"
	backendcore "github.com/oasislabs/developer-gateway/backend/core"
	"github.com/oasislabs/developer-gateway/callback"
	callbackclient "github.com/oasislabs/developer-gateway/callback/client"
	"github.com/oasislabs/developer-gateway/log"
	"github.com/oasislabs/developer-gateway/mqueue"
	"github.com/oasislabs/developer-gateway/rpc"
	"github.com/sirupsen/logrus"
)

// RootLogger is the base logger of the application, all
// loggers used in the gateway should derive from this
var RootLogger = log.NewLogrus(log.LogrusLoggerProperties{
	Level: logrus.WarnLevel,
})

// RootContext is the base logger of the application,
// all contexts created in the gateway should derive
// from this
var RootContext = context.Background()

type Services struct {
	Callback      *callbackclient.Client
	Request       *backendcore.RequestManager
	Backend       backendcore.Client
	Authenticator authcore.Auth
}

type ServiceFactories struct {
	MailboxFactory        mqueue.MailboxFactory
	CallbacksFactory      callback.ClientFactory
	BackendClientFactory  backend.ClientFactory
	BackendRequestManager backend.RequestManagerFactory
	AuthFactory           auth.Factory
}

func setDefaultFactories(factories *ServiceFactories) *ServiceFactories {
	if factories == nil {
		factories = &ServiceFactories{}
	}
	if factories.MailboxFactory == nil {
		factories.MailboxFactory = mqueue.NewMailbox
	}
	if factories.CallbacksFactory == nil {
		factories.CallbacksFactory = callback.NewClient
	}
	if factories.BackendClientFactory == nil {
		factories.BackendClientFactory = backend.NewBackendClient
	}
	if factories.BackendRequestManager == nil {
		factories.BackendRequestManager = backend.NewRequestManagerWithDeps
	}
	if factories.AuthFactory == nil {
		factories.AuthFactory = auth.NewAuth
	}

	return factories
}

// InitLogger initializes the static RootLogger with the
// provided configuration. This should be called before
// RootLogger is used
func InitLogger(config *LoggingConfig) {
	props := log.LogrusLoggerProperties{
		Level: logrus.DebugLevel,
	}

	switch config.Level {
	case "debug":
		props.Level = logrus.DebugLevel
	case "info":
		props.Level = logrus.InfoLevel
	case "warn":
		props.Level = logrus.WarnLevel
	default:
		props.Level = logrus.DebugLevel
	}

	RootLogger = log.NewLogrus(props)
}

func NewServicesWithFactories(ctx context.Context, config *Config, factories *ServiceFactories) (*Services, error) {
	factories = setDefaultFactories(factories)
	mqueue, err := factories.MailboxFactory.New(ctx, mqueue.Services{Logger: RootLogger}, &config.MailboxConfig)
	if err != nil {
		return nil, err
	}

	callbacks, err := factories.CallbacksFactory.New(ctx, &callback.ClientServices{
		Logger: RootLogger,
	}, &config.CallbackConfig)
	if err != nil {
		return nil, err
	}

	client, err := factories.BackendClientFactory.New(ctx, &backend.ClientServices{
		Logger:    RootLogger,
		Callbacks: callbacks,
	}, &config.BackendConfig)
	if err != nil {
		return nil, err
	}

	request, err := factories.BackendRequestManager.New(ctx, &backend.Deps{
		Logger: RootLogger,
		MQueue: mqueue,
		Client: client,
	})
	if err != nil {
		return nil, err
	}

	authenticator, err := factories.AuthFactory.New(&config.AuthConfig)
	if err != nil {
		return nil, err
	}

	return &Services{
		Request:       request,
		Backend:       client,
		Authenticator: authenticator,
		Callback:      callbacks,
	}, nil
}

func NewServices(ctx context.Context, config *Config) (*Services, error) {
	return NewServicesWithFactories(ctx, config, nil)
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

func NewPublicRouter(services *Services) *rpc.HttpRouter {
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

			return authcore.NewHttpMiddlewareAuth(services.Authenticator, RootLogger, jsonHandler)
		}),
	})

	service.BindHandler(service.Services{
		Logger:   RootLogger,
		Client:   services.Request,
		Verifier: authcore.TrustedPayloadVerifier{},
	}, binder)
	event.BindHandler(event.Services{
		Logger:  RootLogger,
		Request: services.Request,
	}, binder)

	return binder.Build()
}
