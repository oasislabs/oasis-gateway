package config

import (
	"errors"
	"strings"

	"github.com/oasislabs/developer-gateway/log"
)

type MailboxProvider string

const (
	MailboxRedisSingle  MailboxProvider = "redis-single"
	MailboxRedisCluster MailboxProvider = "redis-cluster"
	MailboxMem          MailboxProvider = "mem"
)

type MailboxConfig struct {
	Binder
	Provider string
	Mailbox  Mailbox
}

func (c *MailboxConfig) Log(fields log.Fields) {
	fields.Add("mailbox.provider", c.Provider)

	if c.Mailbox != nil {
		c.Mailbox.Log(fields)
	}
}

func (c *MailboxConfig) Configure(flagBinder *FlagBinder) error {
	c.Provider = flagBinder.GetString("mailbox", "provider")
	if len(c.Provider) == 0 {
		return errors.New("mailbox.provider must be set. " +
			"Options are " + string(MailboxMem) +
			", " + string(MailboxRedisSingle) +
			", " + string(MailboxRedisCluster) + ".")
	}

	switch MailboxProvider(c.Provider) {
	case MailboxMem:
		c.Mailbox = &MailboxMemConfig{}
		return c.Mailbox.(*MailboxMemConfig).Configure(flagBinder)
	case MailboxRedisSingle:
		c.Mailbox = &MailboxRedisSingleConfig{}
		return c.Mailbox.(*MailboxRedisSingleConfig).Configure(flagBinder)
	case MailboxRedisCluster:
		c.Mailbox = &MailboxRedisClusterConfig{}
		return c.Mailbox.(*MailboxRedisClusterConfig).Configure(flagBinder)
	default:
		return errors.New("unknown mailbox.provider set. " +
			"Options are " + string(MailboxMem) +
			", " + string(MailboxRedisSingle) +
			", " + string(MailboxRedisCluster) + ".")
	}
}

func (c *MailboxConfig) Bind(flagBinder *FlagBinder) error {
	if err := flagBinder.BindStringFlag("mailbox", "provider", "mem",
		"provider for the mailbox service. "+
			"Options are "+string(MailboxMem)+
			", "+string(MailboxRedisSingle)+
			", "+string(MailboxRedisCluster)+"."); err != nil {
		return err
	}

	if err := (&MailboxRedisSingleConfig{}).Bind(flagBinder); err != nil {
		return err
	}
	if err := (&MailboxRedisClusterConfig{}).Bind(flagBinder); err != nil {
		return err
	}
	if err := (&MailboxMemConfig{}).Bind(flagBinder); err != nil {
		return err
	}

	return nil
}

type Mailbox interface {
	log.Loggable
	Binder
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

func (c *MailboxRedisSingleConfig) Configure(flagBinder *FlagBinder) error {
	c.Addr = flagBinder.GetString("mailbox.redis_single", "addr")
	if len(c.Addr) == 0 {
		return errors.New("mailbox.redis_single.addr must be set")
	}

	return nil
}

func (c *MailboxRedisSingleConfig) Bind(flagBinder *FlagBinder) error {
	return flagBinder.BindStringFlag("mailbox.redis_single", "addr", "127.0.0.1:6379", "redis instance address")
}

type MailboxRedisClusterConfig struct {
	Addrs []string
}

func (c *MailboxRedisClusterConfig) Log(fields log.Fields) {
	fields.Add("mailbox.redis_cluster.addrs", strings.Join(c.Addrs, " - "))
}

func (c *MailboxRedisClusterConfig) ID() MailboxProvider {
	return MailboxRedisCluster
}

func (c *MailboxRedisClusterConfig) Configure(flagBinder *FlagBinder) error {
	return nil
}

func (c *MailboxRedisClusterConfig) Bind(flagBinder *FlagBinder) error {
	return nil
}

type MailboxMemConfig struct{}

func (c *MailboxMemConfig) Log(fields log.Fields) {}

func (c *MailboxMemConfig) ID() MailboxProvider {
	return MailboxMem
}

func (c *MailboxMemConfig) Configure(flagBinder *FlagBinder) error {
	return nil
}

func (c *MailboxMemConfig) Bind(flagBinder *FlagBinder) error {
	return nil
}
