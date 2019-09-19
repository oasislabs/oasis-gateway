package callback

import (
	"fmt"
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
	c.Sync = v.GetBool("callback.wallet_out_of_funds.sync")
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
	cmd.PersistentFlags().Bool("callback.wallet_out_of_funds.sync", false,
		"whether to send the callback synchronously.")

	return nil
}

func (c *WalletOutOfFunds) Log(fields log.Fields) {
	fields.Add("callback.wallet_out_of_funds.enabled", c.Enabled)
	fields.Add("callback.wallet_out_of_funds.method", c.Method)
	fields.Add("callback.wallet_out_of_funds.url", c.URL)
	fields.Add("callback.wallet_out_of_funds.body", c.Body)
	fields.Add("callback.wallet_out_of_funds.queryurl", c.QueryURL)
	fields.Add("callback.wallet_out_of_funds.headers", strings.Join(c.Headers, ","))
	fields.Add("callback.wallet_out_of_funds.sync", c.Sync)
}

type TransactionCommitted struct {
	Callback
}

func (c *TransactionCommitted) Configure(v *viper.Viper) error {
	c.Enabled = v.GetBool("callback.transaction_committed.enabled")
	if !c.Enabled {
		return nil
	}

	c.Method = v.GetString("callback.transaction_committed.method")
	if len(c.Method) == 0 {
		return config.ErrKeyNotSet{Key: "callback.transaction_committed.method"}
	}

	c.URL = v.GetString("callback.transaction_committed.url")
	if len(c.URL) == 0 {
		return config.ErrKeyNotSet{Key: "callback.transaction_committed.url"}
	}

	c.Body = v.GetString("callback.transaction_committed.body")
	c.QueryURL = v.GetString("callback.transaction_committed.queryurl")
	c.Headers = v.GetStringSlice("callback.transaction_committed.headers")
	c.Sync = v.GetBool("callback.transaction_committed.sync")
	return nil
}

func (c *TransactionCommitted) Bind(v *viper.Viper, cmd *cobra.Command) error {
	cmd.PersistentFlags().Bool("callback.transaction_committed.enabled", false,
		"enables the transaction_committed callback. This callback will be sent by the"+
			"gateway when the provided wallet has run out of funds to execute a transaction.")
	cmd.PersistentFlags().String("callback.transaction_committed.method", "",
		"http method on the request for the callback.")
	cmd.PersistentFlags().String("callback.transaction_committed.url", "",
		"http url for the callback.")
	cmd.PersistentFlags().String("callback.transaction_committed.body", "",
		"http body for the callback.")
	cmd.PersistentFlags().String("callback.transaction_committed.queryurl", "",
		"http query url for the callback.")
	cmd.PersistentFlags().StringSlice("callback.transaction_committed.headers", nil,
		"http headers for the callback.")
	cmd.PersistentFlags().Bool("callback.transaction_committed.sync", false,
		"whether to send the callback synchronously.")

	return nil
}

func (c *TransactionCommitted) Log(fields log.Fields) {
	fields.Add("callback.transaction_committed.enabled", c.Enabled)
	fields.Add("callback.transaction_committed.method", c.Method)
	fields.Add("callback.transaction_committed.url", c.URL)
	fields.Add("callback.transaction_committed.body", c.Body)
	fields.Add("callback.transaction_committed.queryurl", c.QueryURL)
	fields.Add("callback.transaction_committed.headers", strings.Join(c.Headers, ","))
	fields.Add("callback.transaction_committed.sync", c.Sync)
}

type WalletReachedFundsThreshold struct {
	Enabled   bool
	Sync      bool
	Method    string
	URL       string
	Body      string
	QueryURL  string
	Headers   []string
	Threshold uint64
}

func (c *WalletReachedFundsThreshold) Configure(v *viper.Viper) error {
	c.Enabled = v.GetBool("callback.wallet_reached_funds_threshold.enabled")
	if !c.Enabled {
		return nil
	}

	c.Method = v.GetString("callback.wallet_reached_funds_threshold.method")
	if len(c.Method) == 0 {
		return config.ErrKeyNotSet{Key: "callback.wallet_reached_funds_threshold.method"}
	}

	c.URL = v.GetString("callback.wallet_reached_funds_threshold.url")
	if len(c.URL) == 0 {
		return config.ErrKeyNotSet{Key: "callback.wallet_reached_funds_threshold.url"}
	}

	c.Body = v.GetString("callback.wallet_reached_funds_threshold.body")
	c.QueryURL = v.GetString("callback.wallet_reached_funds_threshold.queryurl")
	c.Headers = v.GetStringSlice("callback.wallet_reached_funds_threshold.headers")
	c.Sync = v.GetBool("callback.wallet_reached_funds_threshold.sync")
	i := v.GetInt64("callback.wallet_reached_funds_threshold.threshold")
	if i < 0 {
		return config.ErrInvalidValue{
			Key:          "callback.wallet_reached_funds_threshold.threshold",
			InvalidValue: fmt.Sprintf("%d", i),
			Values:       []string{},
		}
	}

	c.Threshold = uint64(i)

	return nil
}

func (c *WalletReachedFundsThreshold) Bind(v *viper.Viper, cmd *cobra.Command) error {
	cmd.PersistentFlags().Bool("callback.wallet_reached_funds_threshold.enabled", false,
		"enables the wallet_reached_funds_threshold callback. This callback will be sent by the"+
			"gateway when the provided wallet has run out of funds to execute a transaction.")
	cmd.PersistentFlags().String("callback.wallet_reached_funds_threshold.method", "",
		"http method on the request for the callback.")
	cmd.PersistentFlags().String("callback.wallet_reached_funds_threshold.url", "",
		"http url for the callback.")
	cmd.PersistentFlags().String("callback.wallet_reached_funds_threshold.body", "",
		"http body for the callback.")
	cmd.PersistentFlags().String("callback.wallet_reached_funds_threshold.queryurl", "",
		"http query url for the callback.")
	cmd.PersistentFlags().StringSlice("callback.wallet_reached_funds_threshold.headers", nil,
		"http headers for the callback.")
	cmd.PersistentFlags().Bool("callback.wallet_reached_funds_threshold.sync", false,
		"whether to send the callback synchronously.")
	cmd.PersistentFlags().Uint64("callback.wallet_reached_funds_threshold.threshold", 0,
		"sets the lower threshold to trigger the callback")

	return nil
}

func (c *WalletReachedFundsThreshold) Log(fields log.Fields) {
	fields.Add("callback.wallet_reached_funds_threshold.enabled", c.Enabled)
	fields.Add("callback.wallet_reached_funds_threshold.method", c.Method)
	fields.Add("callback.wallet_reached_funds_threshold.url", c.URL)
	fields.Add("callback.wallet_reached_funds_threshold.body", c.Body)
	fields.Add("callback.wallet_reached_funds_threshold.queryurl", c.QueryURL)
	fields.Add("callback.wallet_reached_funds_threshold.headers", strings.Join(c.Headers, ","))
	fields.Add("callback.wallet_reached_funds_threshold.sync", c.Sync)
}

type Callback struct {
	Enabled  bool
	Sync     bool
	Method   string
	URL      string
	Body     string
	QueryURL string
	Headers  []string
}

type Config struct {
	TransactionCommitted        TransactionCommitted
	WalletOutOfFunds            WalletOutOfFunds
	WalletReachedFundsThreshold WalletReachedFundsThreshold
}

func (c *Config) Configure(v *viper.Viper) error {
	if err := c.TransactionCommitted.Configure(v); err != nil {
		return err
	}
	if err := c.WalletOutOfFunds.Configure(v); err != nil {
		return err
	}
	if err := c.WalletReachedFundsThreshold.Configure(v); err != nil {
		return err
	}
	return nil
}

func (c *Config) Bind(v *viper.Viper, cmd *cobra.Command) error {
	if err := c.TransactionCommitted.Bind(v, cmd); err != nil {
		return err
	}
	if err := c.WalletOutOfFunds.Bind(v, cmd); err != nil {
		return err
	}
	if err := c.WalletReachedFundsThreshold.Bind(v, cmd); err != nil {
		return err
	}
	return nil
}

func (c *Config) Log(fields log.Fields) {
	c.TransactionCommitted.Log(fields)
	c.WalletOutOfFunds.Log(fields)
	c.WalletReachedFundsThreshold.Log(fields)
}
