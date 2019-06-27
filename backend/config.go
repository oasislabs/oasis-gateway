package backend

import (
	"errors"

	"github.com/oasislabs/developer-gateway/config"
	"github.com/oasislabs/developer-gateway/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type BackendProvider string

const (
	BackendEthereum BackendProvider = "ethereum"
	BackendEkiden   BackendProvider = "ekiden"
)

func (m BackendProvider) String() string {
	return string(m)
}

type Config struct {
	Provider      BackendProvider
	BackendConfig BackendConfig
}

func (c *Config) Log(fields log.Fields) {
	fields.Add("backend.provider", c.Provider)

	if c.BackendConfig != nil {
		c.BackendConfig.Log(fields)
	}
}

func (c *Config) Configure(v *viper.Viper) error {
	c.Provider = BackendProvider(v.GetString("backend.provider"))
	if len(c.Provider) == 0 {
		return config.ErrKeyNotSet{Key: "backend.provider"}
	}

	switch c.Provider {
	case BackendEthereum:
		c.BackendConfig = &EthereumConfig{}
		return c.BackendConfig.(*EthereumConfig).Configure(v)
	case BackendEkiden:
		return config.ErrNotImplemented{
			Key:   "backend.provider",
			Value: BackendEkiden.String(),
		}
	default:
		return config.ErrInvalidValue{
			Key:          "backend.provider",
			InvalidValue: c.Provider.String(),
			Values:       []string{BackendEthereum.String(), BackendEkiden.String()},
		}
	}
}

func (c *Config) Bind(v *viper.Viper, cmd *cobra.Command) error {
	cmd.PersistentFlags().String("backend.provider", "ethereum",
		"provider for the mailbox service. "+
			"Options are "+BackendEthereum.String()+
			", "+BackendEkiden.String()+".")

	if err := (&EthereumConfig{}).Bind(v, cmd); err != nil {
		return err
	}

	return nil
}

type BackendConfig interface {
	log.Loggable
	config.Binder
	ID() BackendProvider
}

type EthereumConfig struct {
	URL          string
	WalletConfig WalletConfig
}

func (c *EthereumConfig) Log(fields log.Fields) {
	fields.Add("eth.url", c.URL)
}

func (c *EthereumConfig) Configure(v *viper.Viper) error {
	c.URL = v.GetString("eth.url")
	if len(c.URL) == 0 {
		return errors.New("eth.url must be set")
	}

	return c.WalletConfig.Configure(v)
}

func (c *EthereumConfig) ID() BackendProvider {
	return BackendEthereum
}

func (c *EthereumConfig) Bind(v *viper.Viper, cmd *cobra.Command) error {
	cmd.PersistentFlags().String("eth.url", "", "url for the eth endpoint")
	return c.WalletConfig.Bind(v, cmd)
}

// WalletConfig holds the configuration of a single wallet
type WalletConfig struct {
	// PrivateKeys for the wallet
	PrivateKeys []string
}

func (c *WalletConfig) Log(fields log.Fields) {
	// do not log the private keys themselves
	fields.Add("eth.wallet.private_keys", len(c.PrivateKeys))
}

func (c *WalletConfig) Configure(v *viper.Viper) error {
	c.PrivateKeys = v.GetStringSlice("eth.wallet.private_keys")

	if len(c.PrivateKeys) == 0 {
		return errors.New("eth.wallet.private_keys must be set")
	}

	for _, key := range c.PrivateKeys {
		if len(key) == 0 {
			return errors.New("eth.wallet.private_keys cannot have empty keys")
		}
	}

	return nil
}

func (c *WalletConfig) Bind(v *viper.Viper, cmd *cobra.Command) error {
	cmd.PersistentFlags().StringSlice("eth.wallet.private_keys", []string{}, "private keys for the wallet")
	return nil
}
