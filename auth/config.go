package auth

import (
	"github.com/oasislabs/developer-gateway/config"
	"github.com/oasislabs/developer-gateway/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type AuthProvider string

const (
	AuthInsecure = "insecure"
	AuthOauth    = "oauth"
)

// Config sets the configuration for the authentication
// mechanism to use
type Config struct {
	Providers []AuthProvider
}

func (c *Config) Log(fields log.Fields) {
	fields.Add("auth.provider", c.Providers)
}

func (c *Config) Configure(v *viper.Viper) error {
	providers := v.GetStringSlice("auth.provider")
	for _, provider := range providers {
		c.Providers = append(c.Providers, AuthProvider(provider))
	}
	if len(c.Providers) < len(providers) {
		return config.ErrKeyNotSet{Key: "auth.provider"}
	}

	return nil
}

func (c *Config) Bind(v *viper.Viper, cmd *cobra.Command) error {
	cmd.PersistentFlags().StringSlice("auth.provider", []string{"insecure"}, "provider for request authentication")
	return nil
}
