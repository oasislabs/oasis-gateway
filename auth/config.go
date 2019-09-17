package auth

import (
	"plugin"
	"strings"

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
	var names []string

	for _, provider := range c.Providers {
		names = append(names, provider.Name())
	}

	fields.Add("auth.provider", strings.Join(names, ", "))
}

func (c *Config) Configure(v *viper.Viper) error {
	if c.Providers == nil {
		c.Providers = make([]core.Auth, 0)
	}

	providers := v.GetStringSlice("auth.provider")
	for _, provider := range providers {
		auth := newAuthSingle(AuthProvider(provider))
		if auth == nil {
			return config.ErrKeyNotSet{Key: "auth.provider"}
		}
		c.Providers = append(c.Providers, auth)
	}

	providers = v.GetStringSlice("auth.plugin")
	for _, provider := range providers {
		plug, err := plugin.Open(provider)
		if err != nil {
			return config.ErrInvalidValue{Key: "auth.provider", InvalidValue: provider}
		}
		symbol, err := plug.Lookup("Auth")
		if err != nil {
			return config.ErrInvalidValue{Key: "auth.provider", InvalidValue: provider}
		}
		auth, ok := symbol.(core.Auth)
		if !ok {
			return config.ErrInvalidValue{Key: "auth.provider", InvalidValue: provider}
		}
		c.Providers = append(c.Providers, auth)
	}

	return nil
}

func (c *Config) Bind(v *viper.Viper, cmd *cobra.Command) error {
	cmd.PersistentFlags().StringSlice("auth.provider", []string{"insecure"}, "providers for request authentication")
	cmd.PersistentFlags().StringSlice("auth.plugin", []string{}, "plugins for request authentication")
	return nil
}
