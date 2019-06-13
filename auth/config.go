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
	Provider AuthProvider
}

func (c *Config) Log(fields log.Fields) {
	fields.Add("auth.provider", c.Provider)
}

func (c *Config) Configure(v *viper.Viper) error {
	c.Provider = AuthProvider(v.GetString("auth.provider"))
	if len(c.Provider) == 0 {
		return config.ErrKeyNotSet{Key: "auth.provider"}
	}

	return nil
}

func (c *Config) Bind(v *viper.Viper, cmd *cobra.Command) error {
	cmd.PersistentFlags().String("auth.provider", "insecure", "provider for request authentication")
	return nil
}
