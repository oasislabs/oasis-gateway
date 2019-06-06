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
	"github.com/spf13/pflag"
)

func publicServer(provider config.Provider) {
	conf := provider.Get()
	bindConfig := conf.BindPublic
	httpInterface := bindConfig.HttpInterface
	httpPort := bindConfig.HttpPort

	services, err := gateway.NewServices(gateway.RootContext, provider.Get(), gateway.Factories{
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
		MaxHeaderBytes: bindConfig.HttpMaxHeaderBytes,
	}

	gateway.RootLogger.Info(gateway.RootContext, "listening to port", log.MapFields{
		"call_type": "HttpPublicListenAttempt",
		"port":      httpPort,
		"interface": httpInterface,
	})
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

func privateServer(provider config.Provider) {
	conf := provider.Get()
	bindConfig := conf.BindPrivate
	httpInterface := bindConfig.HttpInterface
	httpPort := bindConfig.HttpPort

	router := gateway.NewPrivateRouter()
	s := &http.Server{
		Addr:           fmt.Sprintf("%s:%d", httpInterface, httpPort),
		Handler:        router,
		ReadTimeout:    time.Duration(bindConfig.HttpReadTimeoutMs) * time.Millisecond,
		WriteTimeout:   time.Duration(bindConfig.HttpWriteTimeoutMs) * time.Millisecond,
		MaxHeaderBytes: bindConfig.HttpMaxHeaderBytes,
	}

	gateway.RootLogger.Info(gateway.RootContext, "listening to port", log.MapFields{
		"call_type": "HttpPrivateListenAttempt",
		"port":      httpPort,
		"interface": httpInterface,
	})
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

func main() {
	var (
		configFile string
	)

	pflag.StringVar(&configFile, "config",
		"cmd/gateway/config/testing.toml",
		"configuration file for the gateway")
	pflag.Parse()

	provider, err := config.ParseSimpleConfig(configFile)
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}

	var wg sync.WaitGroup

	wg.Add(2)
	go func() {
		publicServer(provider)
		wg.Done()
	}()

	go func() {
		privateServer(provider)
		wg.Done()
	}()

	wg.Wait()
}
