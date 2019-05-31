package main

import (
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/oasislabs/developer-gateway/gateway"
	"github.com/oasislabs/developer-gateway/gateway/config"
	"github.com/oasislabs/developer-gateway/log"
	"github.com/spf13/pflag"
)

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

	bindConfig := provider.Get().Bind
	err = bindConfig.Verify(config.BindConfig{
		HttpInterface:      "127.0.0.1",
		HttpPort:           1234,
		HttpReadTimeoutMs:  10000,
		HttpWriteTimeoutMs: 10000,
		HttpMaxHeaderBytes: 1 << 10,
	})
	if err != nil {
		gateway.RootLogger.Fatal(gateway.RootContext, "failed to verify bind config", log.MapFields{
			"err": err.Error(),
		})
		os.Exit(1)
	}

	httpInterface := bindConfig.HttpInterface
	httpPort := bindConfig.HttpPort

	services, err := gateway.NewServices(gateway.RootContext, provider.Get(), gateway.Factories{
		EthClientFactory: gateway.EthClientFactoryFunc(gateway.NewEthClient),
	})
	if err != nil {
		gateway.RootLogger.Fatal(gateway.RootContext, "failed to initialize services", log.MapFields{
			"err": err.Error(),
		})
		os.Exit(1)
	}

	router := gateway.NewRouter(services)

	s := &http.Server{
		Addr:           fmt.Sprintf("%s:%d", httpInterface, httpPort),
		Handler:        router,
		ReadTimeout:    time.Duration(bindConfig.HttpReadTimeoutMs) * time.Millisecond,
		WriteTimeout:   time.Duration(bindConfig.HttpWriteTimeoutMs) * time.Millisecond,
		MaxHeaderBytes: bindConfig.HttpMaxHeaderBytes,
	}

	if err := s.ListenAndServe(); err != nil {
		gateway.RootLogger.Fatal(gateway.RootContext, "http server failed to listen", log.MapFields{
			"err": err.Error(),
		})
		os.Exit(1)
	}
}
