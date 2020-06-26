package redis

import (
	"context"
	"encoding/json"

	"github.com/go-redis/redis"
	"github.com/oasislabs/oasis-gateway/log"
	"github.com/oasislabs/oasis-gateway/metrics"
	"github.com/oasislabs/oasis-gateway/mqueue/core"
	"github.com/oasislabs/oasis-gateway/stats"
)

const (
	insert   string = "insert"
	retrieve string = "retrieve"
	discard  string = "discard"
	next     string = "next"
	remove   string = "remove"
	exists   string = "exists"
)

// Client is the interface to the redis client used implementing
// the methods used by the MQueue implementation
type Client interface {
	Eval(script string, keys []string, args ...interface{}) *redis.Cmd
	Exists(key ...string) *redis.IntCmd
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
	client  Client
	logger  log.Logger
	tracker *stats.MethodTracker
	metrics *metrics.DatabaseMetrics
}

// NewClusterMQueue creates a new instance of a redis client
// ready to be used against a redis cluster
func NewClusterMQueue(props ClusterProps) (*MQueue, error) {
	logger := props.Logger.ForClass("mqueue/redis", "MQueue")
	c := redis.NewClusterClient(&redis.ClusterOptions{
		Addrs: props.Addrs,
	})

	return &MQueue{
		client:  c,
		logger:  logger,
		tracker: stats.NewMethodTracker(insert, retrieve, discard, next, remove, exists),
		metrics: metrics.NewDefaultDatabaseMetrics("redis"),
	}, nil
}

// NewSingleMQueue creates a new instance of a redis client
// ready to be used against a single instance of redis
func NewSingleMQueue(props SingleInstanceProps) (*MQueue, error) {
	logger := props.Logger.ForClass("mqueue/redis", "MQueue")
	c := redis.NewClient(&redis.Options{
		Addr: props.Addr,
	})

	return &MQueue{
		client:  c,
		logger:  logger,
		tracker: stats.NewMethodTracker(insert, retrieve, discard, next, remove),
		metrics: metrics.NewDefaultDatabaseMetrics("oasis-gateway-redis"),
	}, nil
}

func (m *MQueue) Name() string {
	return "mqueue.redis.MQueue"
}

func (m *MQueue) Stats() stats.Metrics {
	return m.tracker.Stats()
}

func (m *MQueue) exec(ctx context.Context, cmd command) (interface{}, error) {
	return m.client.Eval(string(cmd.Op()), cmd.Keys(), cmd.Args()...).Result()
}

func (m *MQueue) Insert(ctx context.Context, req core.InsertRequest) error {
	timer := m.metrics.DatabaseTimer(insert)
	defer timer.ObserveDuration()

	err := m.insert(ctx, req)
	if err != nil {
		m.metrics.DatabaseCounter(insert, "fail").Inc()
	} else {
		m.metrics.DatabaseCounter(insert, "success").Inc()
	}

	return err
}

func (m *MQueue) insert(ctx context.Context, req core.InsertRequest) error {
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
	timer := m.metrics.DatabaseTimer(retrieve)
	defer timer.ObserveDuration()

	els, err := m.retrieve(ctx, req)
	if err != nil {
		m.metrics.DatabaseCounter(retrieve, "fail").Inc()
		return core.Elements{}, err
	}

	m.metrics.DatabaseCounter(retrieve, "success").Inc()
	return els, nil
}

func (m *MQueue) retrieve(ctx context.Context, req core.RetrieveRequest) (core.Elements, error) {
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

		if !offsetSet {
			// the offset needs to be set to the first element in the window regardless
			// of whether it is set or not.
			offset = decoded.Offset
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
			Offset: decoded.Offset,
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
	timer := m.metrics.DatabaseTimer(discard)
	defer timer.ObserveDuration()

	err := m.discard(ctx, req)
	if err != nil {
		m.metrics.DatabaseCounter(discard, "fail").Inc()
	} else {
		m.metrics.DatabaseCounter(discard, "success").Inc()
	}

	return err
}

func (m *MQueue) discard(ctx context.Context, req core.DiscardRequest) error {
	v, err := m.exec(ctx, discardRequest{
		Key:          req.Key,
		Offset:       req.Offset,
		Count:        req.Count,
		KeepPrevious: req.KeepPrevious,
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
	timer := m.metrics.DatabaseTimer(next)
	defer timer.ObserveDuration()

	offset, err := m.next(ctx, req)
	if err != nil {
		m.metrics.DatabaseCounter(next, "fail").Inc()
		return 0, err
	}
	m.metrics.DatabaseCounter(next, "success").Inc()
	return offset, nil
}

func (m *MQueue) next(ctx context.Context, req core.NextRequest) (uint64, error) {
	v, err := m.exec(ctx, nextRequest{
		Key: req.Key,
	})
	if err != nil {
		return 0, ErrRedisExec{Cause: err}
	}

	return uint64(v.(int64)), nil
}

func (m *MQueue) Remove(ctx context.Context, req core.RemoveRequest) error {
	timer := m.metrics.DatabaseTimer(remove)
	defer timer.ObserveDuration()

	err := m.remove(ctx, req)
	if err != nil {
		m.metrics.DatabaseCounter(remove, "fail").Inc()
	} else {
		m.metrics.DatabaseCounter(remove, "success").Inc()
	}

	return err
}

func (m *MQueue) Exists(ctx context.Context, req core.ExistsRequest) (bool, error) {
	timer := m.metrics.DatabaseTimer(remove)
	defer timer.ObserveDuration()

	b, err := m.exists(ctx, req)
	if err != nil {
		m.metrics.DatabaseCounter(remove, "fail").Inc()
		return false, err
	}
	m.metrics.DatabaseCounter(remove, "success").Inc()
	return b, nil
}

func (m *MQueue) exists(ctx context.Context, req core.ExistsRequest) (bool, error) {
	v, err := m.client.Exists(req.Key).Result()
	return v == 1, err
}

func (m *MQueue) remove(ctx context.Context, req core.RemoveRequest) error {
	v, err := m.exec(ctx, removeRequest{
		Key: req.Key,
	})

	if err != nil {
		return ErrRedisExec{Cause: err}
	}

	if v.(int64) == 0 {
		return ErrQueueNotFound
	}

	return nil
}
