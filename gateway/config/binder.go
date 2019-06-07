package config

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type Binder interface {
	Bind(*viper.Viper, *cobra.Command) error
	Configure(*viper.Viper) error
}
