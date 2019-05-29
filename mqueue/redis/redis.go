package redis

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/go-redis/redis"
	"github.com/oasislabs/developer-gateway/mqueue/core"
)

type Client interface {
	Eval(script string, keys []string, args ...interface{}) *redis.Cmd
}

// MQueueProps are the properties used to create an instance
// of an MQueue
type MQueueProps struct {
	// Addrs is a seed list of host:port addresses of cluster nodes
	Addrs []string

	// ScriptPath is the path to the script that provides extra
	// functionality to call redis
	ScriptPath string
}

// MQueue implements the messaging queue functionality required
// from the mqueue package using Redis as a backend
type MQueue struct {
	client     Client
	scriptHash string
}

func NewMQueue(props MQueueProps) (*MQueue, error) {
	f, err := os.Open(props.ScriptPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open script file to path %s with error %s", props.ScriptPath, err.Error())
	}

	c := redis.NewClusterClient(&redis.ClusterOptions{
		Addrs: props.Addrs,
	})

	p, err := ioutil.ReadAll(f)
	if err != nil {
		return nil, fmt.Errorf("failed to read script file to path %s with error %s", props.ScriptPath, err.Error())
	}

	hash, err := c.ScriptLoad(string(p)).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to load script file to path %s with error %s", props.ScriptPath, err.Error())
	}

	return &MQueue{client: c, scriptHash: hash}, nil
}

func (m *MQueue) exec(ctx context.Context, cmd command) (interface{}, error) {
	return m.client.Eval(string(cmd.Op()), cmd.Keys(), cmd.Args()).Result()
}

func (m *MQueue) Insert(ctx context.Context, req core.InsertRequest) error {
	serialized, err := json.Marshal(req.Element.Value)
	if err != nil {
		return ErrSerialize{Cause: err}
	}

	v, err := m.exec(ctx, insertRequest{
		Key:     req.Key,
		Offset:  req.Element.Offset,
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
	v, err := m.exec(ctx, retrieveRequest{
		Key:    req.Key,
		Offset: req.Offset,
		Count:  req.Count,
	})

	if err != nil {
		return core.Elements{}, ErrRedisExec{Cause: err}
	}

	fmt.Println(v)
	return core.Elements{}, errors.New("not implemented")
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

	return v.(uint64), nil
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
