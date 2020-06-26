package tests

import (
	"fmt"
	"os"

	"github.com/oasislabs/oasis-gateway/config"
	"github.com/oasislabs/oasis-gateway/gateway"
	"github.com/oasislabs/oasis-gateway/log"
)

var Config *gateway.Config

func init() {
	if len(os.Getenv("OASIS_DG_CONFIG_PATH")) == 0 {
		// set a reasonable default for this
		fmt.Println(
			`OASIS_DG_CONFIG_PATH not set. OASIS_DG_CONFIG_PATH can be used to set the configuration file to use for running the tests. Using config/dev.toml by default`)
		os.Setenv("OASIS_DG_CONFIG_PATH", "config/dev.toml")
	}

	if err := Initialize(); err != nil {
		fmt.Println("Failed to initialize test ", err.Error())
		os.Exit(1)
	}
}

func Initialize() error {
	Config = &gateway.Config{}
	parser, err := config.Generate(Config)
	if err != nil {
		return err
	}

	err = parser.Parse()
	if err != nil {
		return err
	}

	gateway.InitLogger(&Config.LoggingConfig)
	gateway.RootLogger.Info(gateway.RootContext, "bind public configuration parsed", log.MapFields{
		"callType": "BindPublicConfigParseSuccess",
	}, &Config.BindPublicConfig)
	gateway.RootLogger.Info(gateway.RootContext, "bind private configuration parsed", log.MapFields{
		"callType": "BindPrivateConfigParseSuccess",
	}, &Config.BindPrivateConfig)
	gateway.RootLogger.Info(gateway.RootContext, "backend configuration parsed", log.MapFields{
		"callType": "BackendConfigParseSuccess",
	}, &Config.BackendConfig)
	gateway.RootLogger.Info(gateway.RootContext, "mailbox configuration parsed", log.MapFields{
		"callType": "MailboxConfigParseSuccess",
	}, &Config.MailboxConfig)
	gateway.RootLogger.Info(gateway.RootContext, "auth config configuration parsed", log.MapFields{
		"callType": "AuthConfigParseSuccess",
	}, &Config.AuthConfig)
	gateway.RootLogger.Info(gateway.RootContext, "callback config configuration parsed", log.MapFields{
		"callType": "CallbackConfigParseSuccess",
	}, &Config.CallbackConfig)
	gateway.RootLogger.Info(gateway.RootContext, "metrics config configuration parsed", log.MapFields{
		"callType": "MetricsConfigParseSuccess",
	}, &Config.MetricsConfig)

	return nil
}
