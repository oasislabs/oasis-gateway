package redis

import (
	"context"
	"encoding/json"
	"strconv"

	"github.com/go-redis/redis"
	"github.com/oasislabs/developer-gateway/log"
	"github.com/oasislabs/developer-gateway/mqueue/core"
)

type Client interface {
	Eval(script string, keys []string, args ...interface{}) *redis.Cmd
}

type Props struct {
	Context context.Context
	Logger  log.Logger
}

type ClusterProps struct {
	Props

	// Addrs is a seed list of host:post for the redis
	// cluster instances
	Addrs []string
}

type SingleInstanceProps struct {
	Props

	// Addr is the address of the redis instance used to connect
	Addr string
}

// MQueue implements the messaging queue functionality required
// from the mqueue package using Redis as a backend
type MQueue struct {
	client Client
	logger log.Logger
}

// NewClusterMQueue creates a new instance of a redis client
// ready to be used against a redis cluster
func NewClusterMQueue(props ClusterProps) (*MQueue, error) {
	logger := props.Logger.ForClass("mqueue/redis", "MQueue")
	c := redis.NewClusterClient(&redis.ClusterOptions{
		Addrs: props.Addrs,
	})

	return &MQueue{client: c, logger: logger}, nil
}

// NewSingleMQueue creates a new instance of a redis client
// ready to be used against a single instance of redis
func NewSingleMQueue(props SingleInstanceProps) (*MQueue, error) {
	logger := props.Logger.ForClass("mqueue/redis", "MQueue")
	c := redis.NewClient(&redis.Options{
		Addr: props.Addr,
	})

	return &MQueue{client: c, logger: logger}, nil
}

func (m *MQueue) exec(ctx context.Context, cmd command) (interface{}, error) {
	return m.client.Eval(string(cmd.Op()), cmd.Keys(), cmd.Args()...).Result()
}

func (m *MQueue) Insert(ctx context.Context, req core.InsertRequest) error {
	serialized, err := json.Marshal(req.Element.Value)
	if err != nil {
		return ErrSerialize{Cause: err}
	}

	v, err := m.exec(ctx, insertRequest{
		Key:     req.Key,
		Offset:  req.Element.Offset,
		Type:    req.Element.Type,
		Content: string(serialized),
	})

	if err != nil {
		return ErrRedisExec{Cause: err}
	}

	if v.(string) != "OK" {
		return ErrOpNotOk
	}

	return nil
}

func (m *MQueue) Retrieve(ctx context.Context, req core.RetrieveRequest) (core.Elements, error) {
	els, err := m.exec(ctx, retrieveRequest{
		Key:    req.Key,
		Offset: req.Offset,
		Count:  req.Count,
	})

	if err != nil {
		return core.Elements{}, ErrRedisExec{Cause: err}
	}

	var res []core.Element
	var offsetSet bool
	var offset uint64

	for _, el := range els.([]interface{}) {
		var decoded redisElement
		if err := json.Unmarshal([]byte(el.(string)), &decoded); err != nil {
			return core.Elements{}, ErrDeserialize{Cause: err}
		}

		elOffset, err := strconv.ParseUint(decoded.Offset, 10, 64)
		if err != nil {
			return core.Elements{}, ErrDeserialize{Cause: err}
		}

		if !offsetSet {
			// the offset needs to be set to the first element in the window regardless
			// of whether it is set or not.
			offset = elOffset
			offsetSet = true
		}

		// just ignore all elements that have not been set yet
		if !decoded.Set {
			continue
		}

		// value is serialized in our redis script as a string, so we need to deserialize
		// the contents of the value as a string
		var value string
		if err := json.Unmarshal([]byte(decoded.Value), &value); err != nil {
			return core.Elements{}, ErrDeserialize{Cause: err}
		}

		res = append(res, core.Element{
			Offset: elOffset,
			Type:   decoded.Type,
			Value:  value,
		})
	}

	return core.Elements{
		Elements: res,
		Offset:   offset,
	}, nil
}

func (m *MQueue) Discard(ctx context.Context, req core.DiscardRequest) error {
	v, err := m.exec(ctx, discardRequest{
		Key:    req.Key,
		Offset: req.Offset,
	})

	if err != nil {
		return ErrRedisExec{Cause: err}
	}

	if v.(string) != "OK" {
		return ErrOpNotOk
	}

	return nil
}

func (m *MQueue) Next(ctx context.Context, req core.NextRequest) (uint64, error) {
	v, err := m.exec(ctx, nextRequest{
		Key: req.Key,
	})
	if err != nil {
		return 0, ErrRedisExec{Cause: err}
	}

	return uint64(v.(int64)), nil
}

func (m *MQueue) Remove(ctx context.Context, req core.RemoveRequest) error {
	v, err := m.exec(ctx, removeRequest{
		Key: req.Key,
	})

	if err != nil {
		return ErrRedisExec{Cause: err}
	}

	if v.(int) == 0 {
		return ErrQueueNotFound
	}

	return nil
}