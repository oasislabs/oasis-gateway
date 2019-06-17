package main

import (
	"fmt"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/oasislabs/developer-gateway/config"
	"github.com/oasislabs/developer-gateway/gateway"
	"github.com/oasislabs/developer-gateway/log"
)

func publicServer(config *gateway.Config) {
	bindConfig := config.BindPublicConfig
	httpInterface := bindConfig.HttpInterface
	httpPort := bindConfig.HttpPort

	services, err := gateway.NewServices(gateway.RootContext, config)
	if err != nil {
		gateway.RootLogger.Fatal(gateway.RootContext, "failed to initialize services", log.MapFields{
			"call_type": "HttpPublicListenFailure",
			"port":      httpPort,
			"interface": httpInterface,
			"err":       err.Error(),
		})
		os.Exit(1)
	}

	router := gateway.NewPublicRouter(services)

	s := &http.Server{
		Addr:           fmt.Sprintf("%s:%d", httpInterface, httpPort),
		Handler:        router,
		ReadTimeout:    time.Duration(bindConfig.HttpReadTimeoutMs) * time.Millisecond,
		WriteTimeout:   time.Duration(bindConfig.HttpWriteTimeoutMs) * time.Millisecond,
		MaxHeaderBytes: int(bindConfig.HttpMaxHeaderBytes),
	}

	gateway.RootLogger.Info(gateway.RootContext, "listening to port", log.MapFields{
		"call_type": "HttpPublicListenAttempt",
		"port":      httpPort,
		"interface": httpInterface,
	})

	if bindConfig.HttpsEnabled {
		if err := s.ListenAndServeTLS(bindConfig.TlsCertificatePath, bindConfig.TlsPrivateKeyPath); err != nil {
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

func privateServer(config *gateway.Config) {
	bindConfig := config.BindPrivateConfig
	httpInterface := bindConfig.HttpInterface
	httpPort := bindConfig.HttpPort

	router := gateway.NewPrivateRouter()
	s := &http.Server{
		Addr:           fmt.Sprintf("%s:%d", httpInterface, httpPort),
		Handler:        router,
		ReadTimeout:    time.Duration(bindConfig.HttpReadTimeoutMs) * time.Millisecond,
		WriteTimeout:   time.Duration(bindConfig.HttpWriteTimeoutMs) * time.Millisecond,
		MaxHeaderBytes: int(bindConfig.HttpMaxHeaderBytes),
	}

	gateway.RootLogger.Info(gateway.RootContext, "listening to port", log.MapFields{
		"call_type": "HttpPrivateListenAttempt",
		"port":      httpPort,
		"interface": httpInterface,
	})

	if bindConfig.HttpsEnabled {
		if err := s.ListenAndServeTLS(bindConfig.TlsCertificatePath, bindConfig.TlsPrivateKeyPath); err != nil {
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

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		publicServer(config)
		wg.Done()
	}()

	go func() {
		privateServer(config)
		wg.Done()
	}()

	wg.Wait()
}
