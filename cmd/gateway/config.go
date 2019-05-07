package main

import (
	"fmt"

	"github.com/spf13/viper"
)

// SimpleConfigProvider is a dumb ConfigProvider that just returns
// the Config instance that it holds
type SimpleConfigProvider struct {
	config Config
}

// Get implementation of ConfigProvider for SimpleConfigProvider
func (c SimpleConfigProvider) Get() Config {
	return c.config
}

// ConfigProvider returns an instance of the configuration
type ConfigProvider interface {
	// Get an instance of the configuration
	Get() Config
}

// WalletConfig holds the configuration of a single wallet
type WalletConfig struct {
	// PrivateKey for the wallet
	PrivateKey string `mapstructure:"private_key"`
}

// EthConfig is the configuration for the ethereum provider
type EthConfig struct {
	// URL for the endpoint that provides ethereum functionality
	URL string `mapstructure:"url"`
}

// Config is the general application's configuration
type Config struct {
	// Wallet is the configured wallet for the application
	Wallet    WalletConfig `mapstructure:"wallet"`
	EthConfig EthConfig    `mapstructure:"eth"`
}

// ParseSimpleConfig parses a configuration file and returns
// a SimpleConfigProvider
func ParseSimpleConfig(filename string) (*SimpleConfigProvider, error) {
	v := viper.New()
	v.SetConfigFile(filename)
	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("Failed to read configuration file with error %s", err.Error())
	}

	var config Config
	if err := v.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("Failed to unmarshal configuration with error %s", err.Error())
	}

	return &SimpleConfigProvider{config: config}, nil
}
