package callback

import (
	"strings"

	"github.com/oasislabs/developer-gateway/config"
	"github.com/oasislabs/developer-gateway/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type WalletOutOfFunds struct {
	Callback
}

func (c *WalletOutOfFunds) Configure(v *viper.Viper) error {
	c.Enabled = v.GetBool("callback.wallet_out_of_funds.enabled")
	if !c.Enabled {
		return nil
	}

	c.Method = v.GetString("callback.wallet_out_of_funds.method")
	if len(c.Method) == 0 {
		return config.ErrKeyNotSet{Key: "callback.wallet_out_of_funds.method"}
	}

	c.URL = v.GetString("callback.wallet_out_of_funds.url")
	if len(c.URL) == 0 {
		return config.ErrKeyNotSet{Key: "callback.wallet_out_of_funds.url"}
	}

	c.Body = v.GetString("callback.wallet_out_of_funds.body")
	c.QueryURL = v.GetString("callback.wallet_out_of_funds.queryurl")
	c.Headers = v.GetStringSlice("callback.wallet_out_of_funds.headers")
	return nil
}

func (c *WalletOutOfFunds) Bind(v *viper.Viper, cmd *cobra.Command) error {
	cmd.PersistentFlags().Bool("callback.wallet_out_of_funds.enabled", false,
		"enables the wallet_out_of_funds callback. This callback will be sent by the"+
			"gateway when the provided wallet has run out of funds to execute a transaction.")
	cmd.PersistentFlags().String("callback.wallet_out_of_funds.method", "",
		"http method on the request for the callback.")
	cmd.PersistentFlags().String("callback.wallet_out_of_funds.url", "",
		"http url for the callback.")
	cmd.PersistentFlags().String("callback.wallet_out_of_funds.body", "",
		"http body for the callback.")
	cmd.PersistentFlags().String("callback.wallet_out_of_funds.queryurl", "",
		"http query url for the callback.")
	cmd.PersistentFlags().StringSlice("callback.wallet_out_of_funds.headers", nil,
		"http headers for the callback.")
	return nil
}

func (c *WalletOutOfFunds) Log(fields log.Fields) {
	fields.Add("callback.wallet_out_of_funds.enabled", c.Enabled)
	fields.Add("callback.wallet_out_of_funds.method", c.Method)
	fields.Add("callback.wallet_out_of_funds.url", c.URL)
	fields.Add("callback.wallet_out_of_funds.body", c.Body)
	fields.Add("callback.wallet_out_of_funds.queryurl", c.QueryURL)
	fields.Add("callback.wallet_out_of_funds.headers", strings.Join(c.Headers, ","))
}

type Callback struct {
	Enabled  bool
	Method   string
	URL      string
	Body     string
	QueryURL string
	Headers  []string
}

type Config struct {
	WalletOutOfFunds WalletOutOfFunds
}

func (c *Config) Configure(v *viper.Viper) error {
	return c.WalletOutOfFunds.Configure(v)
}

func (c *Config) Bind(v *viper.Viper, cmd *cobra.Command) error {
	return c.WalletOutOfFunds.Bind(v, cmd)
}

func (c *Config) Log(fields log.Fields) {
	c.WalletOutOfFunds.Log(fields)
}
