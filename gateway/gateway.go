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

var RootLogger = log.NewLogrus(log.LogrusLoggerProperties{
	Level: logrus.DebugLevel,
})

var RootContext = context.Background()

type Services struct {
	Callback      *callbackclient.Client
	Request       *backendcore.RequestManager
	Backend       backendcore.Client
	Authenticator authcore.Auth
}

func NewServices(ctx context.Context, config *Config) (*Services, error) {
	mqueue, err := mqueue.NewMailbox(ctx, mqueue.Services{Logger: RootLogger}, &config.MailboxConfig)
	if err != nil {
		return nil, err
	}

	callbacks, err := callback.NewClient(ctx, &callback.ClientServices{
		Logger: RootLogger,
	}, &config.CallbackConfig)
	if err != nil {
		return nil, err
	}

	client, err := backend.NewBackendClient(ctx, &backend.ClientServices{
		Logger:    RootLogger,
		Callbacks: callbacks,
	}, &config.BackendConfig)
	if err != nil {
		return nil, err
	}

	request, err := backend.NewRequestManagerWithDeps(ctx, &backend.Deps{
		Logger: RootLogger,
		MQueue: mqueue,
		Client: client,
	})
	if err != nil {
		return nil, err
	}

	authenticator, err := auth.NewAuth(&config.AuthConfig)
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
		Verifier: services.Authenticator,
	}, binder)
	event.BindHandler(event.Services{
		Logger:  RootLogger,
		Request: services.Request,
	}, binder)

	return binder.Build()
}
