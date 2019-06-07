package config

import (
	"fmt"
	"os"
	"path"
	"strings"
)

type ConfigFile struct {
	Path string
}

func (f *ConfigFile) Bind(flagBinder *FlagBinder) error {
	return flagBinder.BindStringFlag("config", "path", "", "sets the configuration file")
}

func (f *ConfigFile) Configure(flagBinder *FlagBinder) error {
	f.Path = flagBinder.GetString("config", "path")
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
	flagBinder.viper.SetConfigType(ext)
	if err := flagBinder.viper.ReadConfig(file); err != nil {
		return fmt.Errorf("failed to read config file %s", err.Error())
	}

	return nil
}
