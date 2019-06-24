package auth

import (
	"plugin"

	"github.com/oasislabs/developer-gateway/auth/core"
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
	Providers []core.Auth
}

func (c *Config) Log(fields log.Fields) {
	fields.Add("auth.provider", c.Providers)
}

func (c *Config) Configure(v *viper.Viper) error {
	providers := v.GetStringSlice("auth.provider")
	for _, provider := range providers {
		auth := newAuthSingle(AuthProvider(provider))
		if auth == nil {
			// try loading as plugin
			plug, err := plugin.Open(provider)
			if err != nil {
				return config.ErrInvalidValue{Key: "auth.provider", InvalidValue: provider}
			}
			symbol, err := plug.Lookup("Auth")
			if err != nil {
				return config.ErrInvalidValue{Key: "auth.provider", InvalidValue: provider}
			}
			var ok bool
			auth, ok = symbol.(core.Auth)
			if !ok {
				return config.ErrInvalidValue{Key: "auth.provider", InvalidValue: provider}
			}
		}
		c.Providers = append(c.Providers, auth)
	}
	if len(c.Providers) < len(providers) {
		return config.ErrKeyNotSet{Key: "auth.provider"}
	}

	return nil
}

func (c *Config) Bind(v *viper.Viper, cmd *cobra.Command) error {
	cmd.PersistentFlags().StringSlice("auth.provider", []string{"insecure"}, "providers for request authentication")
	return nil
}
