package main

import (
	"fmt"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/oasislabs/developer-gateway/gateway"
	"github.com/oasislabs/developer-gateway/gateway/config"
	"github.com/oasislabs/developer-gateway/log"
)

func publicServer(conf *config.Config) {
	bindConfig := conf.BindPublicConfig
	httpInterface := bindConfig.HttpInterface
	httpPort := bindConfig.HttpPort

	services, err := gateway.NewServices(gateway.RootContext, conf, gateway.Factories{
		EthClientFactory: gateway.EthClientFactoryFunc(gateway.NewEthClient),
	})
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

func privateServer(conf *config.Config) {
	bindConfig := conf.BindPrivateConfig
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

	d, err := os.Getwd()
	fmt.Println("current WD: ", d, err)
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
	parser, err := config.Generate()
	if err != nil {
		fmt.Println("Failed to generate configurations: ", err.Error())
		os.Exit(1)
	}

	conf, err := parser.Parse()
	if err != nil {
		fmt.Println("Failed to parse configuration: ", err.Error())
		if err := parser.Usage(); err != nil {
			panic("failed to print usage")
		}
		os.Exit(1)
	}

	gateway.RootLogger.Info(gateway.RootContext, "bind public configuration parsed", log.MapFields{
		"callType": "BindPublicConfigParseSuccess",
	}, &conf.BindPublicConfig)
	gateway.RootLogger.Info(gateway.RootContext, "bind private configuration parsed", log.MapFields{
		"callType": "BindPrivateConfigParseSuccess",
	}, &conf.BindPrivateConfig)
	gateway.RootLogger.Info(gateway.RootContext, "wallet configuration parsed", log.MapFields{
		"callType": "WalletConfigParseSuccess",
	}, &conf.WalletConfig)
	gateway.RootLogger.Info(gateway.RootContext, "eth configuration parsed", log.MapFields{
		"callType": "EthConfigParseSuccess",
	}, &conf.EthConfig)
	gateway.RootLogger.Info(gateway.RootContext, "mailbox configuration parsed", log.MapFields{
		"callType": "MailboxConfigParseSuccess",
	}, &conf.MailboxConfig)
	gateway.RootLogger.Info(gateway.RootContext, "auth config configuration parsed", log.MapFields{
		"callType": "AuthConfigParseSuccess",
	}, &conf.AuthConfig)

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		publicServer(conf)
		wg.Done()
	}()

	go func() {
		privateServer(conf)
		wg.Done()
	}()

	wg.Wait()
}
