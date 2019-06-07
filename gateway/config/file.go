package config

import (
	"fmt"
	"os"
	"path"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type ConfigFile struct {
	Path string
}

func (f *ConfigFile) Bind(v *viper.Viper, cmd *cobra.Command) error {
	cmd.PersistentFlags().String("config.path", "", "sets the configuration file")
	return nil
}

func (f *ConfigFile) Configure(v *viper.Viper) error {
	f.Path = v.GetString("config.path")
	if len(f.Path) == 0 {
		// if not config file is set there is not need to
		// read anything, so we can just return
		return nil
	}

	ext := strings.TrimPrefix(path.Ext(f.Path), ".")
	if ext != "toml" && ext != "yaml" {
		return fmt.Errorf("config file extension must be .toml or .yaml")
	}

	file, err := os.Open(f.Path)
	if err != nil {
		return fmt.Errorf("failed to open config file %s", err.Error())
	}

	defer func() { _ = file.Close() }()
	v.SetConfigType(ext)
	if err := v.ReadConfig(file); err != nil {
		return fmt.Errorf("failed to read config file %s", err.Error())
	}

	return nil
}
