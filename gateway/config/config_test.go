package config

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

func TestBindConfigNoHttps(t *testing.T) {
	cmd := &cobra.Command{}
	v := viper.New()
	c := BindConfig{}

	err := c.Bind("bind_public", v, cmd)
	assert.Nil(t, err)

	v.Set("bind_public.http_interface", "localhost")
	v.Set("bind_public.http_port", 1234)
	v.Set("bind_public.http_interface", "localhost")
	v.Set("bind_public.http_read_timeout_ms", 10000)
	v.Set("bind_public.http_write_timeout_ms", 10000)
	v.Set("bind_public.http_max_header_bytes", 10000)

	err = c.Configure("bind_public", v)
	assert.Nil(t, err)

	assert.Equal(t, BindConfig{
		HttpInterface:      "localhost",
		HttpPort:           1234,
		HttpReadTimeoutMs:  10000,
		HttpWriteTimeoutMs: 10000,
		HttpMaxHeaderBytes: 10000,
	}, c)
}

func TestBindConfigInvalidPort(t *testing.T) {
	cmd := &cobra.Command{}
	v := viper.New()
	c := BindConfig{}

	err := c.Bind("bind_public", v, cmd)
	assert.Nil(t, err)

	v.Set("bind_public.http_interface", "")
	v.Set("bind_public.http_port", -1)
	v.Set("bind_public.http_interface", "localhost")
	v.Set("bind_public.http_read_timeout_ms", 10000)
	v.Set("bind_public.http_write_timeout_ms", 10000)
	v.Set("bind_public.http_max_header_bytes", 10000)

	err = c.Configure("bind_public", v)
	assert.Error(t, err)
}

func TestBindConfigInvalidReadTimeout(t *testing.T) {
	cmd := &cobra.Command{}
	v := viper.New()
	c := BindConfig{}

	err := c.Bind("bind_public", v, cmd)
	assert.Nil(t, err)

	v.Set("bind_public.http_interface", "")
	v.Set("bind_public.http_port", 1234)
	v.Set("bind_public.http_interface", "localhost")
	v.Set("bind_public.http_read_timeout_ms", -1)
	v.Set("bind_public.http_write_timeout_ms", 10000)
	v.Set("bind_public.http_max_header_bytes", 10000)

	err = c.Configure("bind_public", v)
	assert.Error(t, err)
}

func TestBindConfigInvalidWriteTimeout(t *testing.T) {
	cmd := &cobra.Command{}
	v := viper.New()
	c := BindConfig{}

	err := c.Bind("bind_public", v, cmd)
	assert.Nil(t, err)

	v.Set("bind_public.http_interface", "")
	v.Set("bind_public.http_port", 1234)
	v.Set("bind_public.http_interface", "localhost")
	v.Set("bind_public.http_read_timeout_ms", 10000)
	v.Set("bind_public.http_write_timeout_ms", -1)
	v.Set("bind_public.http_max_header_bytes", 10000)

	err = c.Configure("bind_public", v)
	assert.Error(t, err)
}

func TestBindConfigInvalidMaxHeaderBytes(t *testing.T) {
	cmd := &cobra.Command{}
	v := viper.New()
	c := BindConfig{}

	err := c.Bind("bind_public", v, cmd)
	assert.Nil(t, err)

	v.Set("bind_public.http_interface", "")
	v.Set("bind_public.http_port", 1234)
	v.Set("bind_public.http_interface", "localhost")
	v.Set("bind_public.http_read_timeout_ms", 10000)
	v.Set("bind_public.http_write_timeout_ms", 10000)
	v.Set("bind_public.http_max_header_bytes", -1)

	err = c.Configure("bind_public", v)
	assert.Error(t, err)
}

func TestBindConfigHttps(t *testing.T) {
	cmd := &cobra.Command{}
	v := viper.New()
	c := BindConfig{}

	err := c.Bind("bind_public", v, cmd)
	assert.Nil(t, err)

	v.Set("bind_public.http_interface", "localhost")
	v.Set("bind_public.http_port", 1234)
	v.Set("bind_public.http_interface", "localhost")
	v.Set("bind_public.http_read_timeout_ms", 10000)
	v.Set("bind_public.http_write_timeout_ms", 10000)
	v.Set("bind_public.http_max_header_bytes", 10000)
	v.Set("bind_public.https_enabled", true)
	v.Set("bind_public.tls_certificate_path", "cert.pem")
	v.Set("bind_public.tls_private_key_path", "key.pem")

	err = c.Configure("bind_public", v)
	assert.Nil(t, err)

	assert.Equal(t, BindConfig{
		HttpInterface:      "localhost",
		HttpPort:           1234,
		HttpReadTimeoutMs:  10000,
		HttpWriteTimeoutMs: 10000,
		HttpMaxHeaderBytes: 10000,
		HttpsEnabled:       true,
		TlsCertificatePath: "cert.pem",
		TlsPrivateKeyPath:  "key.pem",
	}, c)
}

func TestBindConfigHttpsNoCert(t *testing.T) {
	cmd := &cobra.Command{}
	v := viper.New()
	c := BindConfig{}

	err := c.Bind("bind_public", v, cmd)
	assert.Nil(t, err)

	v.Set("bind_public.http_interface", "localhost")
	v.Set("bind_public.http_port", 1234)
	v.Set("bind_public.http_interface", "localhost")
	v.Set("bind_public.http_read_timeout_ms", 10000)
	v.Set("bind_public.http_write_timeout_ms", 10000)
	v.Set("bind_public.http_max_header_bytes", 10000)
	v.Set("bind_public.https_enabled", true)
	v.Set("bind_public.tls_private_key_path", "key.pem")

	err = c.Configure("bind_public", v)
	assert.Error(t, err)
}

func TestBindConfigHttpsNoPrivateKey(t *testing.T) {
	cmd := &cobra.Command{}
	v := viper.New()
	c := BindConfig{}

	err := c.Bind("bind_public", v, cmd)
	assert.Nil(t, err)

	v.Set("bind_public.http_interface", "localhost")
	v.Set("bind_public.http_port", 1234)
	v.Set("bind_public.http_interface", "localhost")
	v.Set("bind_public.http_read_timeout_ms", 10000)
	v.Set("bind_public.http_write_timeout_ms", 10000)
	v.Set("bind_public.http_max_header_bytes", 10000)
	v.Set("bind_public.https_enabled", true)
	v.Set("bind_public.tls_certificate_path", "cert.pem")

	err = c.Configure("bind_public", v)
	assert.Error(t, err)
}
