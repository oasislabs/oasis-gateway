package log

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type Config struct {
	Level string
}

func (c *Config) Log(fields Fields) {
	fields.Add("logging.level", c.Level)
}

func (c *Config) Configure(v *viper.Viper) error {
	c.Level = v.GetString("logging.level")
	if len(c.Level) == 0 {
		c.Level = "debug"
	}

	return nil
}

func (c *Config) Bind(v *viper.Viper, cmd *cobra.Command) error {
	cmd.PersistentFlags().String("logging.level", "debug",
		"sets the minimum logging level for the logger")
	return nil
}
