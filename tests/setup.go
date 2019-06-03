package tests

import (
	"fmt"
	"io/ioutil"
	"os"

	"github.com/oasislabs/developer-gateway/gateway"
	"github.com/oasislabs/developer-gateway/gateway/config"
	"github.com/oasislabs/developer-gateway/rpc"
	"github.com/oasislabs/developer-gateway/tests/mock"
)

var router *rpc.HttpRouter

func init() {
	test := os.Getenv("OASIS_GATEWAY_TEST")
	if len(test) == 0 {
		fmt.Println("OASIS_GATEWAY_TEST needs to be set to the type of" +
			" tests to run. Options are: 'dev', 'redis_single'")
		os.Exit(1)
	}

	r, err := Initialize(test)
	if err != nil {
		fmt.Println("Failed to initialize test ", err.Error())
		os.Exit(1)
	}

	router = r
}

func Initialize(config string) (*rpc.HttpRouter, error) {
	switch config {
	case "dev":
		return InitializeWithConfig("config/dev.toml")
	case "redis_single":
		return InitializeWithConfig("config/redis_single.toml")
	default:
		return nil, fmt.Errorf("unknown configuration type provided %s", config)
	}
}

func InitializeWithConfig(configFile string) (*rpc.HttpRouter, error) {
	provider, err := config.ParseSimpleConfig(configFile)
	if err != nil {
		return nil, err
	}

	services, err := gateway.NewServices(gateway.RootContext, provider.Get(), gateway.Factories{
		EthClientFactory: gateway.EthClientFactoryFunc(mock.NewMockEthClient),
	})
	if err != nil {
		return nil, err
	}

	gateway.RootLogger.SetOutput(ioutil.Discard)
	return gateway.NewRouter(services), nil
}
