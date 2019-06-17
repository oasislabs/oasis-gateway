package tests

import (
	"fmt"
	"io/ioutil"
	"os"

	"github.com/oasislabs/developer-gateway/config"
	"github.com/oasislabs/developer-gateway/gateway"
	"github.com/oasislabs/developer-gateway/log"
	"github.com/oasislabs/developer-gateway/rpc"
	"github.com/oasislabs/developer-gateway/tests/mock"
)

var router *rpc.HttpRouter

func init() {
	path := os.Getenv("OASIS_DG_CONFIG_PATH")
	if len(path) == 0 {
		path = ".oasis_dev_gateway.toml"
		fmt.Println("Using default configuration location of '.oasis_dev_gateway.toml'")
	}

	r, err := Initialize()
	if err != nil {
		fmt.Println("Failed to initialize test ", err.Error())
		os.Exit(1)
	}

	router = r
}

func Initialize() (*rpc.HttpRouter, error) {
	parser, err := config.Generate(&gateway.Config{})
	if err != nil {
		return nil, err
	}

	err = parser.Parse()
	if err != nil {
		return nil, err
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

	services, err := mock.NewServices(gateway.RootContext, config)
	if err != nil {
		return nil, err
	}

	gateway.RootLogger.SetOutput(ioutil.Discard)
	return gateway.NewPublicRouter(services), nil
}
