package config

import (
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type Binder interface {
	Bind(*FlagBinder) error
	Configure(*FlagBinder) error
}

type FlagBinder struct {
	viper *viper.Viper
	cmd   *cobra.Command
}

// BindStringFlag binds a string flag to be parsed from the
// command line, as an environmetn variable or from a configuration file
func (b FlagBinder) BindStringFlag(namespace, name string, def string, desc string) error {
	envNamespace := strings.Replace(namespace, ".", "_", -1)
	flag := namespace + "." + name
	env := envNamespace + "_" + name
	b.cmd.PersistentFlags().String(flag, def, desc)

	if err := b.viper.BindPFlag(env, b.cmd.PersistentFlags().Lookup(flag)); err != nil {
		return err
	}

	return b.viper.BindPFlag(flag, b.cmd.PersistentFlags().Lookup(flag))
}

// GetString returns the string value while preserving the order defined
// by viper:
//  - command line argument
//  - environment variable
//  - config file
func (b FlagBinder) GetString(namespace, name string) string {
	envNamespace := strings.Replace(namespace, ".", "_", -1)
	flag := namespace + "." + name
	env := envNamespace + "_" + name

	f, err := b.cmd.PersistentFlags().GetString(flag)
	changed := b.cmd.Flags().Changed(flag)
	if err != nil && changed {
		return f
	}

	switch {
	case len(b.viper.GetString(flag)) > 0:
		return b.viper.GetString(flag)
	case len(b.viper.GetString(env)) > 0:
		return b.viper.GetString(env)
	case err == nil:
		return f
	default:
		return ""
	}
}

// BindInt32Flag binds a string flag to be parsed from the
// command line, as an environmetn variable or from a configuration file
func (b FlagBinder) BindInt32Flag(namespace, name string, def int32, desc string) error {
	envNamespace := strings.Replace(namespace, ".", "_", -1)
	flag := namespace + "." + name
	env := envNamespace + "_" + name
	b.cmd.PersistentFlags().Int32(flag, def, desc)

	if err := b.viper.BindPFlag(env, b.cmd.PersistentFlags().Lookup(flag)); err != nil {
		return err
	}
	return b.viper.BindPFlag(flag, b.cmd.PersistentFlags().Lookup(flag))
}

// GetInt32 returns the int32 value while preserving the order defined
// by viper:
//  - command line argument
//  - environment variable
//  - config file
func (b FlagBinder) GetInt32(namespace, name string) int32 {
	envNamespace := strings.Replace(namespace, ".", "_", -1)
	flag := namespace + "." + name
	env := envNamespace + "_" + name

	f, err := b.cmd.PersistentFlags().GetInt32(flag)
	changed := b.cmd.Flags().Changed(flag)
	if err == nil && changed {
		return f
	}

	// NOTE: with this implementation if somebody sets an environment variable
	// to 0 it will be skipped and the default command line value
	// would be used. This may not be the desired behaviour.
	switch {
	case b.viper.GetInt32(flag) != 0:
		return b.viper.GetInt32(flag)
	case b.viper.GetInt32(env) != 0:
		return b.viper.GetInt32(env)
	case err == nil:
		return f
	default:
		return 0
	}
}
