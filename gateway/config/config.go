package config

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/oasislabs/developer-gateway/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// BindConfig is the configuration for binding the exposed APIs
// to the computer network interface
type BindConfig struct {
	HttpInterface      string
	HttpPort           int32
	HttpReadTimeoutMs  int32
	HttpWriteTimeoutMs int32
	HttpMaxHeaderBytes int32
}

func (c *BindConfig) Configure(prefix string, v *viper.Viper) error {
	c.HttpInterface = v.GetString(prefix + ".http_interface")
	if len(c.HttpInterface) == 0 {
		return errors.New(prefix + ".http_interface must be set")
	}

	c.HttpPort = v.GetInt32(prefix + ".http_port")
	if c.HttpPort > 65535 || c.HttpPort < 0 {
		return errors.New(prefix + ".http_port must be an integer between 0 and 65535")
	}

	c.HttpReadTimeoutMs = v.GetInt32(prefix + ".http_read_timeout_ms")
	if c.HttpReadTimeoutMs < 0 {
		return errors.New(prefix + ".http_read_timeout_ms cannot be negative")
	}

	c.HttpWriteTimeoutMs = v.GetInt32(prefix + ".http_write_timeout_ms")
	if c.HttpWriteTimeoutMs < 0 {
		return errors.New(prefix + ".http_write_timeout_ms cannot be negative")
	}

	c.HttpMaxHeaderBytes = v.GetInt32(prefix + ".http_max_header_bytes")
	if c.HttpMaxHeaderBytes < 0 {
		return errors.New(prefix + ".http_max_header_bytes cannot be negative")
	}

	return nil
}

func (c *BindConfig) Bind(prefix string, v *viper.Viper, cmd *cobra.Command) error {
	cmd.PersistentFlags().String(prefix+".http_interface", "127.0.0.1",
		"interface to bind for http")
	cmd.PersistentFlags().Int32(prefix+".http_port", 1234,
		"port to listen to for http")
	cmd.PersistentFlags().Int32(prefix+".http_read_timeout_ms",
		10000, "http read timeout for http interface")
	cmd.PersistentFlags().Int32(prefix+".http_write_timeout_ms",
		10000, "http write timeout for http interface")
	cmd.PersistentFlags().Int32(prefix+".http_max_header_bytes",
		10000, "http max header bytes for http")

	return nil
}

type BindPublicConfig struct {
	BindConfig
}

func (c *BindPublicConfig) Log(fields log.Fields) {
	fields.Add("bind_public.http_interface", c.BindConfig.HttpInterface)
	fields.Add("bind_public.http_port", c.BindConfig.HttpPort)
	fields.Add("bind_public.http_read_timeout_ms", c.BindConfig.HttpReadTimeoutMs)
	fields.Add("bind_public.http_write_timeout_ms", c.BindConfig.HttpWriteTimeoutMs)
	fields.Add("bind_public.http_max_header_bytes", c.BindConfig.HttpMaxHeaderBytes)
}

func (c *BindPublicConfig) Configure(v *viper.Viper) error {
	return c.BindConfig.Configure("bind_public", v)
}

func (c *BindPublicConfig) Bind(v *viper.Viper, cmd *cobra.Command) error {
	return c.BindConfig.Bind("bind_public", v, cmd)
}

type BindPrivateConfig struct {
	BindConfig
}

func (c *BindPrivateConfig) Log(fields log.Fields) {
	fields.Add("bind_private.http_interface", c.BindConfig.HttpInterface)
	fields.Add("bind_private.http_port", c.BindConfig.HttpPort)
	fields.Add("bind_private.http_read_timeout_ms", c.BindConfig.HttpReadTimeoutMs)
	fields.Add("bind_private.http_write_timeout_ms", c.BindConfig.HttpWriteTimeoutMs)
	fields.Add("bind_private.http_max_header_bytes", c.BindConfig.HttpMaxHeaderBytes)
}

func (c *BindPrivateConfig) Name() string {
	return "bind_private"
}

func (c *BindPrivateConfig) Configure(v *viper.Viper) error {
	return c.BindConfig.Configure("bind_private", v)
}

func (c *BindPrivateConfig) Bind(v *viper.Viper, cmd *cobra.Command) error {
	return c.BindConfig.Bind("bind_private", v, cmd)
}

// WalletConfig holds the configuration of a single wallet
type WalletConfig struct {
	// PrivateKey for the wallet
	PrivateKey string
}

func (c *WalletConfig) Log(fields log.Fields) {
	// do not log the private key itself
	fields.Add("wallet.private_key_set", true)
}

func (c *WalletConfig) Configure(v *viper.Viper) error {
	c.PrivateKey = v.GetString("wallet.private_key")
	if len(c.PrivateKey) == 0 {
		return errors.New("wallet.private_key must be set")
	}

	return nil
}

func (c *WalletConfig) Bind(v *viper.Viper, cmd *cobra.Command) error {
	cmd.PersistentFlags().String("wallet.private_key", "", "private key for the wallet")
	return nil
}

// EthConfig is the configuration for the ethereum provider
type EthConfig struct {
	// URL for the endpoint that provides ethereum functionality
	URL string
}

func (c *EthConfig) Log(fields log.Fields) {
	fields.Add("eth.url", c.URL)
}

func (c *EthConfig) Configure(v *viper.Viper) error {
	c.URL = v.GetString("eth.url")
	if len(c.URL) == 0 {
		return errors.New("eth.url must be set")
	}

	return nil
}

func (c *EthConfig) Bind(v *viper.Viper, cmd *cobra.Command) error {
	cmd.PersistentFlags().String("eth.url", "", "url for the eth endpoint")
	return nil
}

// AuthConfig sets the configuration for the authentication
// mechanism to use
type AuthConfig struct {
	Provider string
}

func (c *AuthConfig) Log(fields log.Fields) {
	fields.Add("auth.provider", c.Provider)
}

func (c *AuthConfig) Configure(v *viper.Viper) error {
	c.Provider = v.GetString("auth.provider")
	if len(c.Provider) == 0 {
		return errors.New("auth.provider must be set")
	}

	return nil
}

func (c *AuthConfig) Bind(v *viper.Viper, cmd *cobra.Command) error {
	cmd.PersistentFlags().String("auth.provider", "insecure", "provider for request authentication")
	return nil
}

// Config is the general application's configuration
type Config struct {
	BindPublicConfig  BindPublicConfig
	BindPrivateConfig BindPrivateConfig
	WalletConfig      WalletConfig
	EthConfig         EthConfig
	MailboxConfig     MailboxConfig
	AuthConfig        AuthConfig
}

func (c *Config) Log(fields log.Fields) {
	c.BindPublicConfig.Log(fields)
	c.BindPrivateConfig.Log(fields)
	c.WalletConfig.Log(fields)
	c.EthConfig.Log(fields)
	c.MailboxConfig.Log(fields)
	c.AuthConfig.Log(fields)
}

type Parser struct {
	file   *ConfigFile
	config *Config
	cmd    *cobra.Command
	v      *viper.Viper
}

func (p *Parser) Parse() (*Config, error) {
	if p.cmd.PersistentFlags().Parsed() {
		return nil, errors.New("arguments already parsed")
	}

	if err := p.cmd.PersistentFlags().Parse(os.Args); err != nil {
		return nil, fmt.Errorf("failed to parse flags %s", err.Error())
	}

	// keep file first so that any parameters read from the file are used
	// as defaults for the other flags
	configs := []Binder{p.file, &p.config.BindPublicConfig, &p.config.BindPrivateConfig, &p.config.WalletConfig,
		&p.config.EthConfig, &p.config.MailboxConfig, &p.config.AuthConfig}

	for _, c := range configs {
		if err := c.Configure(p.v); err != nil {
			return nil, err
		}
	}

	return p.config, nil
}

func (p *Parser) Usage() error {
	return p.cmd.Usage()
}

func Generate() (*Parser, error) {
	v := viper.New()
	// sets a default prefix for all environment variables that will be
	// parsed by viper and automatically binds all expected variables
	// from the environment
	v.SetEnvPrefix("OASIS_DG")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	cmd := &cobra.Command{Use: "developer-gateway"}
	config := Config{}
	file := ConfigFile{}
	configs := []Binder{&file, &config.BindPublicConfig, &config.BindPrivateConfig, &config.WalletConfig,
		&config.EthConfig, &config.MailboxConfig, &config.AuthConfig}

	for _, c := range configs {
		if err := c.Bind(v, cmd); err != nil {
			return nil, fmt.Errorf("failed to bind flags %s", err.Error())
		}
	}

	return &Parser{file: &file, config: &config, cmd: cmd, v: v}, nil
}
