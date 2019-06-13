package mqueue

import (
	"context"
	"fmt"

	"github.com/oasislabs/developer-gateway/log"
	"github.com/oasislabs/developer-gateway/mqueue/core"
	"github.com/oasislabs/developer-gateway/mqueue/mem"
	"github.com/oasislabs/developer-gateway/mqueue/redis"
)

type Services struct {
	Logger log.Logger
}

func NewMailbox(ctx context.Context, services Services, config *Config) (core.MQueue, error) {
	if config.MailboxConfig.ID() != config.Provider {
		return nil, ErrBackendConfigConflict
	}

	switch config.MailboxConfig.ID() {
	case MailboxRedisSingle:
		return NewRedisSingleMailbox(ctx, services, config.MailboxConfig.(*MailboxRedisSingleConfig))
	case MailboxRedisCluster:
		return NewRedisClusterMailbox(ctx, services, config.MailboxConfig.(*MailboxRedisClusterConfig))
	case MailboxMem:
		return mem.NewServer(ctx, mem.Services{
			Logger: services.Logger,
		}), nil
	default:
		return nil, ErrUnknownBackend{Backend: config.MailboxConfig.ID().String()}
	}
}

func NewRedisSingleMailbox(
	ctx context.Context,
	services Services,
	config *MailboxRedisSingleConfig,
) (core.MQueue, error) {
	m, err := redis.NewSingleMQueue(redis.SingleInstanceProps{
		Props: redis.Props{
			Context: ctx,
			Logger:  services.Logger,
		},
		Addr: config.Addr,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to start redis mqueue %s", err.Error())
	}
	return m, nil
}

func NewRedisClusterMailbox(
	ctx context.Context,
	services Services,
	config *MailboxRedisClusterConfig,
) (core.MQueue, error) {
	m, err := redis.NewClusterMQueue(redis.ClusterProps{
		Props: redis.Props{
			Context: ctx,
			Logger:  services.Logger,
		},
		Addrs: config.Addrs,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to start redis mqueue %s", err.Error())
	}
	return m, nil
}
