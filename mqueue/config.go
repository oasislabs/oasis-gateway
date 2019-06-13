package mqueue

import (
	"errors"
	"strings"

	"github.com/oasislabs/developer-gateway/config"
	"github.com/oasislabs/developer-gateway/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type MailboxProvider string

const (
	MailboxRedisSingle  MailboxProvider = "redis-single"
	MailboxRedisCluster MailboxProvider = "redis-cluster"
	MailboxMem          MailboxProvider = "mem"
)

func (m MailboxProvider) String() string {
	return string(m)
}

type Config struct {
	Provider      MailboxProvider
	MailboxConfig MailboxConfig
}

func (c *Config) Log(fields log.Fields) {
	fields.Add("mailbox.provider", c.Provider)

	if c.MailboxConfig != nil {
		c.MailboxConfig.Log(fields)
	}
}

func (c *Config) Configure(v *viper.Viper) error {
	c.Provider = MailboxProvider(v.GetString("mailbox.provider"))
	if len(c.Provider) == 0 {
		return config.ErrKeyNotSet{Key: "mailbox.provider"}
	}

	switch c.Provider {
	case MailboxMem:
		c.MailboxConfig = &MailboxMemConfig{}
		return c.MailboxConfig.(*MailboxMemConfig).Configure(v)
	case MailboxRedisSingle:
		c.MailboxConfig = &MailboxRedisSingleConfig{}
		return c.MailboxConfig.(*MailboxRedisSingleConfig).Configure(v)
	case MailboxRedisCluster:
		c.MailboxConfig = &MailboxRedisClusterConfig{}
		return c.MailboxConfig.(*MailboxRedisClusterConfig).Configure(v)
	default:
		return config.ErrInvalidValue{
			Key:          "mailbox.provider",
			InvalidValue: c.Provider.String(),
			Values: []string{
				MailboxRedisSingle.String(),
				MailboxRedisCluster.String(),
				MailboxMem.String(),
			},
		}
	}
}

func (c *Config) Bind(v *viper.Viper, cmd *cobra.Command) error {
	cmd.PersistentFlags().String("mailbox.provider", "mem",
		"provider for the mailbox service. "+
			"Options are "+string(MailboxMem)+
			", "+string(MailboxRedisSingle)+
			", "+string(MailboxRedisCluster)+".")

	if err := (&MailboxRedisSingleConfig{}).Bind(v, cmd); err != nil {
		return err
	}
	if err := (&MailboxRedisClusterConfig{}).Bind(v, cmd); err != nil {
		return err
	}
	if err := (&MailboxMemConfig{}).Bind(v, cmd); err != nil {
		return err
	}

	return nil
}

type MailboxConfig interface {
	log.Loggable
	config.Binder
	ID() MailboxProvider
}

type MailboxRedisSingleConfig struct {
	Addr string
}

func (c *MailboxRedisSingleConfig) Log(fields log.Fields) {
	fields.Add("mailbox.redis_single.addr", c.Addr)
}

func (c *MailboxRedisSingleConfig) ID() MailboxProvider {
	return MailboxRedisSingle
}

func (c *MailboxRedisSingleConfig) Configure(v *viper.Viper) error {
	c.Addr = v.GetString("mailbox.redis_single.addr")
	if len(c.Addr) == 0 {
		return errors.New("mailbox.redis_single.addr must be set")
	}

	return nil
}

func (c *MailboxRedisSingleConfig) Bind(v *viper.Viper, cmd *cobra.Command) error {
	cmd.PersistentFlags().String("mailbox.redis_single.addr", "127.0.0.1:6379", "redis instance address")
	return nil
}

type MailboxRedisClusterConfig struct {
	Addrs []string
}

func (c *MailboxRedisClusterConfig) Log(fields log.Fields) {
	fields.Add("mailbox.redis_cluster.addrs", strings.Join(c.Addrs, ","))
}

func (c *MailboxRedisClusterConfig) ID() MailboxProvider {
	return MailboxRedisCluster
}

func (c *MailboxRedisClusterConfig) Configure(v *viper.Viper) error {
	c.Addrs = v.GetStringSlice("mailbox.redis_cluster.addrs")
	if len(c.Addrs) == 0 {
		return errors.New("mailbox.redis_cluster.addrs must be set")
	}

	return nil
}

func (c *MailboxRedisClusterConfig) Bind(v *viper.Viper, cmd *cobra.Command) error {
	cmd.PersistentFlags().StringArray(
		"mailbox.redis_cluster.addrs",
		[]string{"127.0.0.1:6379"},
		"array of addresses for bootstrap redis instances in the cluster")
	return nil
}

type MailboxMemConfig struct{}

func (c *MailboxMemConfig) Log(fields log.Fields) {}

func (c *MailboxMemConfig) ID() MailboxProvider {
	return MailboxMem
}

func (c *MailboxMemConfig) Configure(v *viper.Viper) error {
	return nil
}

func (c *MailboxMemConfig) Bind(v *viper.Viper, cmd *cobra.Command) error {
	return nil
}
