package redis

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/go-redis/redis"
	"github.com/oasislabs/developer-gateway/mqueue/core"
)

type Client interface {
	EvalSha(sha1 string, keys []string, args ...interface{}) *redis.Cmd
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

func (m *MQueue) Insert(ctx context.Context, req core.InsertRequest) error {
	return errors.New("not implemented")
}

func (m *MQueue) Retrieve(ctx context.Context, req core.RetrieveRequest) error {
	return errors.New("not implemented")
}

func (m *MQueue) Discard(ctx context.Context, req core.DiscardRequest) error {
	return errors.New("not implemented")
}

func (m *MQueue) Next(ctx context.Context, req core.NextRequest) (uint64, error) {
	v, err := m.client.EvalSha(m.scriptHash, []string{req.Key}).Result()
	if err != nil {
		return 0, err
	}

	return v.(uint64), nil
}

func (m *MQueue) Remove(ctx context.Context, req core.RemoveRequest) error {
	return errors.New("not implemented")
}
