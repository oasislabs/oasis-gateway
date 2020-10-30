package main

import (
	"fmt"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/oasislabs/oasis-gateway/config"
	"github.com/oasislabs/oasis-gateway/gateway"
	"github.com/oasislabs/oasis-gateway/log"
	"github.com/oasislabs/oasis-gateway/metrics"
	"github.com/oasislabs/oasis-gateway/rpc"
)

func publicServer(config *gateway.BindPublicConfig, router *rpc.HttpRouter) {
	httpInterface := config.HttpInterface
	httpPort := config.HttpPort

	s := &http.Server{
		Addr:           fmt.Sprintf("%s:%d", httpInterface, httpPort),
		Handler:        router,
		ReadTimeout:    time.Duration(config.HttpReadTimeoutMs) * time.Millisecond,
		WriteTimeout:   time.Duration(config.HttpWriteTimeoutMs) * time.Millisecond,
		MaxHeaderBytes: int(config.HttpMaxHeaderBytes),
	}

	gateway.RootLogger.Info(gateway.RootContext, "listening to port", log.MapFields{
		"call_type": "HttpPublicListenAttempt",
		"port":      httpPort,
		"interface": httpInterface,
	})

	if config.HttpsEnabled {
		if err := s.ListenAndServeTLS(config.TlsCertificatePath, config.TlsPrivateKeyPath); err != nil {
			gateway.RootLogger.Fatal(gateway.RootContext, "http server failed to listen", log.MapFields{
				"call_type": "HttpPublicListenFailure",
				"port":      httpPort,
				"interface": httpInterface,
				"err":       err.Error(),
			})
			os.Exit(1)
		}

	} else {
		if err := s.ListenAndServe(); err != nil {
			gateway.RootLogger.Fatal(gateway.RootContext, "http server failed to listen", log.MapFields{
				"call_type": "HttpPublicListenFailure",
				"port":      httpPort,
				"interface": httpInterface,
				"err":       err.Error(),
			})
			os.Exit(1)
		}
	}
}

func privateServer(config *gateway.BindPrivateConfig, router *rpc.HttpRouter) {
	httpInterface := config.HttpInterface
	httpPort := config.HttpPort

	s := &http.Server{
		Addr:           fmt.Sprintf("%s:%d", httpInterface, httpPort),
		Handler:        router,
		ReadTimeout:    time.Duration(config.HttpReadTimeoutMs) * time.Millisecond,
		WriteTimeout:   time.Duration(config.HttpWriteTimeoutMs) * time.Millisecond,
		MaxHeaderBytes: int(config.HttpMaxHeaderBytes),
	}

	gateway.RootLogger.Info(gateway.RootContext, "listening to port", log.MapFields{
		"call_type": "HttpPrivateListenAttempt",
		"port":      httpPort,
		"interface": httpInterface,
	})

	if config.HttpsEnabled {
		if err := s.ListenAndServeTLS(config.TlsCertificatePath, config.TlsPrivateKeyPath); err != nil {
			gateway.RootLogger.Fatal(gateway.RootContext, "http server failed to listen", log.MapFields{
				"call_type": "HttpPrivateListenFailure",
				"port":      httpPort,
				"interface": httpInterface,
				"err":       err.Error(),
			})
			os.Exit(1)
		}

	} else {
		if err := s.ListenAndServe(); err != nil {
			gateway.RootLogger.Fatal(gateway.RootContext, "http server failed to listen", log.MapFields{
				"call_type": "HttpPrivateListenFailure",
				"port":      httpPort,
				"interface": httpInterface,
				"err":       err.Error(),
			})
			os.Exit(1)
		}
	}
}

func metricsServer(config *metrics.MetricsConfig) {
	if s, err := metrics.New(
		config,
		"oasis-gateway",
		gateway.RootLogger,
	); err != nil {
		gateway.RootLogger.Error(gateway.RootContext, "oasis-gateway: failed to initialize instrumentation", log.MapFields{
			"err": err.Error(),
		})
	} else {
		s.StartInstrumentation()
	}
}

func main() {
	parser, err := config.Generate(&gateway.Config{})
	if err != nil {
		fmt.Println("Failed to generate configurations: ", err.Error())
		os.Exit(1)
	}

	err = parser.Parse()
	if err != nil {
		fmt.Println("Failed to parse configuration: ", err.Error())
		if err := parser.Usage(); err != nil {
			panic("failed to print usage")
		}
		os.Exit(1)
	}

	config := parser.Config.(*gateway.Config)
	gateway.InitLogger(&config.LoggingConfig)

	gateway.RootLogger.Info(gateway.RootContext, "bind public configuration parsed", log.MapFields{
		"callType": "BindPublicConfigParseSuccess",
	}, &config.BindPublicConfig)
	gateway.RootLogger.Info(gateway.RootContext, "bind private configuration parsed", log.MapFields{
		"callType": "BindPrivateConfigParseSuccess",
	}, &config.BindPrivateConfig)
	gateway.RootLogger.Info(gateway.RootContext, "backend configuration parsed", log.MapFields{
		"callType": "BackendConfigParseSuccess",
	}, &config.BackendConfig)
	gateway.RootLogger.Info(gateway.RootContext, "mailbox configuration parsed", log.MapFields{
		"callType": "MailboxConfigParseSuccess",
	}, &config.MailboxConfig)
	gateway.RootLogger.Info(gateway.RootContext, "auth config configuration parsed", log.MapFields{
		"callType": "AuthConfigParseSuccess",
	}, &config.AuthConfig)
	gateway.RootLogger.Info(gateway.RootContext, "callback config configuration parsed", log.MapFields{
		"callType": "CallbackConfigParseSuccess",
	}, &config.CallbackConfig)
	gateway.RootLogger.Info(gateway.RootContext, "metrics config configuration parsed", log.MapFields{
		"callType": "MetricsConfigParseSuccess",
	}, &config.MetricsConfig)

	var wg sync.WaitGroup
	wg.Add(3)

	group, err := gateway.NewServiceGroup(gateway.RootContext, config)
	if err != nil {
		gateway.RootLogger.Fatal(gateway.RootContext, "failed to initialize services", log.MapFields{
			"call_type": "HttpPublicListenFailure",
			"err":       err.Error(),
		})
		os.Exit(1)
	}

	routers := gateway.NewRouters(config, group)

	go func() {
		publicServer(&config.BindPublicConfig, routers.Public)
		wg.Done()
	}()

	go func() {
		privateServer(&config.BindPrivateConfig, routers.Private)
		wg.Done()
	}()

	go func() {
		metricsServer(&config.MetricsConfig)
		wg.Done()
	}()

	wg.Wait()
}
