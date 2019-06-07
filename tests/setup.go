package tests

import (
	"fmt"
	"io/ioutil"
	"os"

	"github.com/oasislabs/developer-gateway/gateway"
	"github.com/oasislabs/developer-gateway/gateway/config"
	"github.com/oasislabs/developer-gateway/log"
	"github.com/oasislabs/developer-gateway/rpc"
	"github.com/oasislabs/developer-gateway/tests/mock"
)

var router *rpc.HttpRouter

func init() {
	path := os.Getenv("OASIS_DG_CONFIG_PATH")
	if len(path) == 0 {
		fmt.Println("OASIS_DG_CONFIG_PATH not set. It must be set to a configuration file ")
		os.Exit(1)
	}

	r, err := Initialize()
	if err != nil {
		fmt.Println("Failed to initialize test ", err.Error())
		os.Exit(1)
	}

	router = r
}

func Initialize() (*rpc.HttpRouter, error) {
	parser, err := config.Generate()
	if err != nil {
		return nil, err
	}

	conf, err := parser.Parse()
	if err != nil {
		return nil, err
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

	services, err := gateway.NewServices(gateway.RootContext, conf, gateway.Factories{
		EthClientFactory: gateway.EthClientFactoryFunc(mock.NewMockEthClient),
	})
	if err != nil {
		return nil, err
	}

	gateway.RootLogger.SetOutput(ioutil.Discard)
	return gateway.NewPublicRouter(services), nil
}
