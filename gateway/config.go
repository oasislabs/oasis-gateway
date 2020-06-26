package gateway

import (
	"errors"
	"math"

	"github.com/oasislabs/oasis-gateway/auth"
	"github.com/oasislabs/oasis-gateway/backend"
	"github.com/oasislabs/oasis-gateway/callback"
	"github.com/oasislabs/oasis-gateway/config"
	"github.com/oasislabs/oasis-gateway/log"
	"github.com/oasislabs/oasis-gateway/metrics"
	"github.com/oasislabs/oasis-gateway/mqueue"
	"github.com/oasislabs/oasis-gateway/rpc"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// Config is the general application's configuration
type Config struct {
	BindPublicConfig  BindPublicConfig
	BindPrivateConfig BindPrivateConfig
	BackendConfig     backend.Config
	MailboxConfig     mqueue.Config
	AuthConfig        auth.Config
	CallbackConfig    callback.Config
	LoggingConfig     LoggingConfig
	MetricsConfig     metrics.MetricsConfig
}

func (c *Config) Use() string {
	return "oasis-gateway"
}

func (c *Config) EnvPrefix() string {
	return "OASIS_DG"
}

func (c *Config) Binders() []config.Binder {
	return []config.Binder{
		&c.BindPublicConfig,
		&c.BindPrivateConfig,
		&c.BackendConfig,
		&c.MailboxConfig,
		&c.AuthConfig,
		&c.CallbackConfig,
		&c.LoggingConfig,
		&c.MetricsConfig,
	}
}

func (c *Config) Log(fields log.Fields) {
	c.BindPublicConfig.Log(fields)
	c.BindPrivateConfig.Log(fields)
	c.BackendConfig.Log(fields)
	c.MailboxConfig.Log(fields)
	c.AuthConfig.Log(fields)
	c.CallbackConfig.Log(fields)
	c.LoggingConfig.Log(fields)
	c.MetricsConfig.Log(fields)
}

// BindConfig is the configuration for binding the exposed APIs
// to the computer network interface
type BindConfig struct {
	HttpInterface      string
	HttpPort           int32
	HttpReadTimeoutMs  int32
	HttpWriteTimeoutMs int32
	HttpMaxHeaderBytes int32
	HttpsEnabled       bool
	TlsCertificatePath string
	TlsPrivateKeyPath  string
	MaxBodyBytes       uint
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

	c.MaxBodyBytes = uint(v.GetInt32(prefix + ".max_body_bytes"))
	if c.MaxBodyBytes > math.MaxInt32 {
		return errors.New(prefix + ".max_body_bytes cannot be negative")
	}

	c.HttpsEnabled = v.GetBool(prefix + ".https_enabled")
	c.TlsCertificatePath = v.GetString(prefix + ".tls_certificate_path")
	c.TlsPrivateKeyPath = v.GetString(prefix + ".tls_private_key_path")

	if c.HttpsEnabled {
		if len(c.TlsCertificatePath) == 0 || len(c.TlsPrivateKeyPath) == 0 {
			return errors.New(prefix + ".tls_certificate_path and " + prefix + ".tls_private_key_path " +
				"must be set if " + prefix + ".https_enabled is set")
		}
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
	cmd.PersistentFlags().Bool(prefix+".https_enabled",
		false, "if set the interface will listen with https. If this option is "+
			"set, then "+prefix+".tls_certificate_path and "+prefix+
			".tls_private_key_path must be set as well")
	cmd.PersistentFlags().Int32(prefix+".max_body_bytes", 1<<16,
		"sets the maximum size for a request body. Any request received"+
			" with a greater body will be rejected")
	cmd.PersistentFlags().String(prefix+".tls_certificate_path",
		"", "path to the tls certificate for https")
	cmd.PersistentFlags().String(prefix+".tls_private_key_path",
		"", "path to the private key for https")

	return nil
}

type BindPublicConfig struct {
	BindConfig
	rpc.HttpCorsPreProcessorProps
}

func (c *BindPublicConfig) Log(fields log.Fields) {
	fields.Add("bind_public.http_interface", c.BindConfig.HttpInterface)
	fields.Add("bind_public.http_port", c.BindConfig.HttpPort)
	fields.Add("bind_public.http_read_timeout_ms", c.BindConfig.HttpReadTimeoutMs)
	fields.Add("bind_public.http_write_timeout_ms", c.BindConfig.HttpWriteTimeoutMs)
	fields.Add("bind_public.http_max_header_bytes", c.BindConfig.HttpMaxHeaderBytes)
	fields.Add("bind_public.https_enabled", c.BindConfig.HttpsEnabled)
	fields.Add("bind_public.max_body_bytes", c.BindConfig.MaxBodyBytes)
	fields.Add("bind_public.tls_certificate_path", c.BindConfig.TlsCertificatePath)
	fields.Add("bind_public.tls_private_key_path", c.BindConfig.TlsPrivateKeyPath)
	fields.Add("bind_public.http_cors.enabled", c.HttpCorsPreProcessorProps.Enabled)
	fields.Add("bind_public.http_cors.allowed_origins", c.HttpCorsPreProcessorProps.AllowedOrigins)
	fields.Add("bind_public.http_cors.allowed_methods", c.HttpCorsPreProcessorProps.AllowedMethods)
	fields.Add("bind_public.http_cors.allowed_headers", c.HttpCorsPreProcessorProps.AllowedHeaders)
	fields.Add("bind_public.http_cors.exposed_headers", c.HttpCorsPreProcessorProps.ExposedHeaders)
	fields.Add("bind_public.http_cors.max_age", c.HttpCorsPreProcessorProps.MaxAge)
	fields.Add("bind_public.http_cors.allowed_credentials", c.HttpCorsPreProcessorProps.AllowCredentials)
}

func (c *BindPublicConfig) Configure(v *viper.Viper) error {
	if err := c.BindConfig.Configure("bind_public", v); err != nil {
		return err
	}

	c.HttpCorsPreProcessorProps.Enabled = v.GetBool("bind_public.http_cors.enabled")
	c.HttpCorsPreProcessorProps.AllowedOrigins = v.GetStringSlice("bind_public.http_cors.allowed_origins")
	c.HttpCorsPreProcessorProps.AllowedMethods = v.GetStringSlice("bind_public.http_cors.allowed_methods")
	c.HttpCorsPreProcessorProps.AllowedHeaders = v.GetStringSlice("bind_public.http_cors.allowed_headers")
	c.HttpCorsPreProcessorProps.ExposedHeaders = v.GetStringSlice("bind_public.http_cors.exposed_headers")
	c.HttpCorsPreProcessorProps.MaxAge = v.GetInt("bind_public.http_cors.max_age")
	c.HttpCorsPreProcessorProps.AllowCredentials = v.GetBool("bind_public.http_cors.allowed_credentials")

	return nil
}

func (c *BindPublicConfig) Bind(v *viper.Viper, cmd *cobra.Command) error {
	if err := c.BindConfig.Bind("bind_public", v, cmd); err != nil {
		return err
	}

	cmd.PersistentFlags().Bool("bind_public.http_cors.enabled", false,
		"if set to true the public port will do CORS handling")
	cmd.PersistentFlags().StringSlice("bind_public.http_cors.allowed_origins", []string{"*"},
		"allowed origins for CORS")
	cmd.PersistentFlags().StringSlice("bind_public.http_cors.allowed_methods", nil,
		"allowed methods for CORS")
	cmd.PersistentFlags().StringSlice("bind_public.http_cors.allowed_headers", nil,
		"allowed headers for CORS")
	cmd.PersistentFlags().StringSlice("bind_public.http_cors.exposed_headers", nil,
		"exposed headers for CORS")
	cmd.PersistentFlags().Int("bind_public.http_cors.max_age", -1,
		"exposed headers for CORS")
	cmd.PersistentFlags().Bool("bind_public.http_cors.allowed_credentials", true,
		"whether credentials are allowed when using CORS")

	return nil
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
	fields.Add("bind_public.max_body_bytes", c.BindConfig.MaxBodyBytes)
	fields.Add("bind_private.https_enabled", c.BindConfig.HttpsEnabled)
	fields.Add("bind_private.tls_certificate_path", c.BindConfig.TlsCertificatePath)
	fields.Add("bind_private.tls_private_key_path", c.BindConfig.TlsPrivateKeyPath)
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

type LoggingConfig struct {
	Level string
}

func (c *LoggingConfig) Log(fields log.Fields) {
	fields.Add("logging.level", c.Level)
}

func (c *LoggingConfig) Configure(v *viper.Viper) error {
	c.Level = v.GetString("logging.level")
	if len(c.Level) == 0 {
		c.Level = "debug"
	}

	return nil
}

func (c *LoggingConfig) Bind(v *viper.Viper, cmd *cobra.Command) error {
	cmd.PersistentFlags().String("logging.level", "debug",
		"sets the minimum logging level for the logger")
	return nil
}
