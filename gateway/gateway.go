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
	mqueuecore "github.com/oasislabs/developer-gateway/mqueue/core"
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

type ServiceGroup struct {
	Mailbox       mqueuecore.MQueue
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

func NewServiceGroupWithFactories(ctx context.Context, config *Config, factories *ServiceFactories) (*ServiceGroup, error) {
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
	authenticator.SetLogger(RootLogger)

	return &ServiceGroup{
		Mailbox:       mqueue,
		Request:       request,
		Backend:       client,
		Authenticator: authenticator,
		Callback:      callbacks,
	}, nil
}

func NewServiceGroup(ctx context.Context, config *Config) (*ServiceGroup, error) {
	return NewServiceGroupWithFactories(ctx, config, nil)
}

// Routers holds the routers available to the application
type Routers struct {
	Public  *rpc.HttpRouter
	Private *rpc.HttpRouter
}

func NewRouters(group *ServiceGroup) *Routers {
	services := NewServices()
	services.Add(group.Mailbox)
	services.Add(group.Callback)
	services.Add(group.Request)
	services.Add(group.Backend)
	services.Add(group.Authenticator)
	services.Add(RuntimeService{})

	var routers Routers
	routers.Public = NewPublicRouter(group)
	services.Add(HttpRouterService{
		name:   "PublicRouter",
		router: routers.Public,
	})

	routers.Private = NewPrivateRouter(services, group)

	return &routers
}

func NewPrivateRouter(services Services, group *ServiceGroup) *rpc.HttpRouter {
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

	health.BindHandler(&health.Deps{Collector: services}, binder)

	return binder.Build()
}

func NewPublicRouter(group *ServiceGroup) *rpc.HttpRouter {
	binder := rpc.NewHttpBinder(rpc.HttpBinderProperties{
		Encoder: rpc.JsonEncoder{},
		Logger:  RootLogger,
		HandlerFactory: rpc.HttpHandlerFactoryFunc(func(factory rpc.EntityFactory, handler rpc.Handler) rpc.HttpMiddleware {
			jsonHandler := rpc.NewHttpJsonHandler(rpc.HttpJsonHandlerProperties{
				Limit:   1 << 22,
				Handler: handler,
				Logger:  RootLogger,
				Factory: factory,
			})

			return authcore.NewHttpMiddlewareAuth(group.Authenticator, RootLogger, jsonHandler)
		}),
	})

	service.BindHandler(service.Services{
		Logger:   RootLogger,
		Client:   group.Request,
		Verifier: group.Authenticator,
	}, binder)
	event.BindHandler(event.Services{
		Logger: RootLogger,
		Client: group.Request,
	}, binder)

	return binder.Build()
}
