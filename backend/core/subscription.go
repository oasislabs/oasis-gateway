package core

import (
	"context"
	stderr "errors"
	"fmt"
	"sync"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/oasislabs/oasis-gateway/errors"
	"github.com/oasislabs/oasis-gateway/log"
	mqueue "github.com/oasislabs/oasis-gateway/mqueue/core"
	"github.com/oasislabs/oasis-gateway/stats"
)

type subscription struct {
	ctx    context.Context
	logger log.Logger
	c      <-chan interface{}
	done   chan<- subscriptionEndEvent
	stop   chan interface{}
	key    string
	mqueue mqueue.MQueue
	wg     sync.WaitGroup
}

type subscriptionProps struct {
	Context context.Context
	Logger  log.Logger
	MQueue  mqueue.MQueue
	Key     string
	Done    chan<- subscriptionEndEvent
	C       <-chan interface{}
}

func newSubscription(props subscriptionProps) *subscription {
	if props.Context == nil {
		panic("Context must be set")
	}
	if props.Logger == nil {
		panic("Logger must be set")
	}
	if props.Done == nil {
		panic("Done must be set")
	}
	if len(props.Key) == 0 {
		panic("subscription key must be set")
	}
	if props.MQueue == nil {
		panic("mqueue must be set")
	}

	return &subscription{
		ctx:    props.Context,
		logger: props.Logger.ForClass("backend/core", "subscription"),
		c:      props.C,
		done:   props.Done,
		stop:   make(chan interface{}),
		key:    props.Key,
		mqueue: props.MQueue,
		wg:     sync.WaitGroup{},
	}
}

func (s *subscription) Stop() {
	close(s.stop)
	s.wg.Wait()
}

func (s *subscription) Start() {
	defer func() {
		err := s.mqueue.Remove(context.Background(), mqueue.RemoveRequest{Key: s.key})
		if err != nil {
			s.logger.Warn(s.ctx, "failed to remove messaging queue", log.MapFields{
				"call_type": "SubscriptionExitFailure",
				"key":       s.key,
			})
		} else {
			s.logger.Debug(s.ctx, "", log.MapFields{
				"call_type": "SubscriptionExitSuccess",
				"key":       s.key,
			})
		}

		s.wg.Done()
	}()

	for {
		select {
		case <-s.ctx.Done():
			return
		case _, ok := <-s.stop:
			if !ok {
				return
			}
		case ev, ok := <-s.c:
			if !ok {
				return
			}

			// TODO(stan); when the subscription fails to insert elements into
			// the queue, the subscription should be closed. In that case,
			// we should define a mechanism to report the errors back to the client

			id, err := s.mqueue.Next(s.ctx, mqueue.NextRequest{Key: s.key})
			if err != nil {
				s.logger.Warn(s.ctx, "failed to find next resource for event", log.MapFields{
					"call_type": "InsertSubscriptionEventFailure",
					"key":       s.key,
					"err":       err.Error(),
				})
				continue
			}

			data, ok := ev.(types.Log)
			if !ok {
				s.logger.Warn(s.ctx, "received event of unexpected type", log.MapFields{
					"call_type": "InsertSubscriptionEventFailure",
					"key":       s.key,
					"type":      fmt.Sprintf("%+v", ev),
				})
				continue
			}

			var topics []string
			for _, topic := range data.Topics {
				topics = append(topics, topic.Hex())
			}

			el, err := makeElement(DataEvent{
				ID:     id,
				Data:   hexutil.Encode(data.Data),
				Topics: topics,
			}, id)
			if err != nil {
				s.logger.Warn(s.ctx, "failed to serialize event", log.MapFields{
					"call_type": "InsertSubscriptionEventFailure",
					"key":       s.key,
					"type":      fmt.Sprintf("%+v", ev),
					"err":       err.Error(),
				})
				continue
			}

			if err := s.mqueue.Insert(s.ctx, mqueue.InsertRequest{Key: s.key, Element: el}); err != nil {
				s.logger.Warn(s.ctx, "failed to insert event to resource", log.MapFields{
					"call_type": "InsertSubscriptionEventFailure",
					"key":       s.key,
					"err":       err.Error(),
				})
			}
		}
	}
}

type subscriptionEndEvent struct {
	Key   string
	Error error
}

type createSubscriptionRequest struct {
	Context context.Context
	Key     string
	Err     chan<- errors.Err
	C       <-chan interface{}
}

type destroySubscriptionRequest struct {
	Context context.Context
	Key     string
	Err     chan<- errors.Err
}

type existsSubscriptionRequest struct {
	Context context.Context
	Key     string
	Out     chan<- bool
}

type statsRequest struct {
	Context context.Context
	Out     chan<- stats.Metrics
}

// SubscriptionManagerProps properties used to create the
// behaviour of the manager and the subscriptions created
type SubscriptionManagerProps struct {
	// Context used by the manager and that can be used
	// to signal a cancellation
	Context context.Context

	// Logger used by the manager and its subscriptions
	Logger log.Logger

	// Mqueue is the messaging queue used to keep the
	// stream of events so that the client can retrieve
	// those events later on
	MQueue mqueue.MQueue
}

// SubscriptionManager manages the lifetime
// of a group of subscriptions
type SubscriptionManager struct {
	ctx     context.Context
	logger  log.Logger
	done    chan subscriptionEndEvent
	req     chan interface{}
	subs    map[string]*subscription
	mqueue  mqueue.MQueue
	metrics SubscriptionMetrics
}

type SubscriptionMetrics struct {
	SubscriptionCount      uint64
	SubscriptionCurrent    uint64
	TotalSubscriptionCount uint64
}

// NewSubscriptionManager creates a new subscription manager
func NewSubscriptionManager(props SubscriptionManagerProps) *SubscriptionManager {
	m := SubscriptionManager{
		ctx:     props.Context,
		logger:  props.Logger.ForClass("backend/core", "SubscriptionManager"),
		done:    make(chan subscriptionEndEvent),
		req:     make(chan interface{}),
		subs:    make(map[string]*subscription),
		mqueue:  props.MQueue,
		metrics: SubscriptionMetrics{},
	}

	go m.startLoop()
	return &m
}

func (m *SubscriptionManager) incrSubscriptions() {
	m.metrics.SubscriptionCount++
	m.metrics.TotalSubscriptionCount++
}

func (m *SubscriptionManager) decrSubscriptions() {
	m.metrics.SubscriptionCount--
}

func (m *SubscriptionManager) startLoop() {
	defer func() {
		for _, sub := range m.subs {
			sub.Stop()
			m.remove(sub.key)
		}
		close(m.done)
		close(m.req)
	}()

	for {
		select {
		case <-m.ctx.Done():
			return
		case ev := <-m.done:
			m.remove(ev.Key)
		case req := <-m.req:
			m.handleRequest(req)
		}
	}
}

func (m *SubscriptionManager) handleRequest(req interface{}) {
	switch req := req.(type) {
	case createSubscriptionRequest:
		m.create(req)
	case destroySubscriptionRequest:
		m.destroy(req)
	case existsSubscriptionRequest:
		m.exists(req)
	case statsRequest:
		m.stats(req)
	default:
		panic("received unknown request")
	}
}

func (m *SubscriptionManager) stats(req statsRequest) {
	// subscriptionCount must be equal to currentSubscriptions, otherwise
	// there must be a bug somewhere
	req.Out <- stats.Metrics{
		"subscriptionCount":      m.metrics.SubscriptionCount,
		"totalSubscriptionCount": m.metrics.TotalSubscriptionCount,
		"currentSubscriptions":   uint64(len(m.subs)),
	}
	close(req.Out)
}

func (m *SubscriptionManager) exists(req existsSubscriptionRequest) {
	defer close(req.Out)
	_, ok := m.subs[req.Key]
	req.Out <- ok
}

func (m *SubscriptionManager) create(req createSubscriptionRequest) {
	defer close(req.Err)

	_, ok := m.subs[req.Key]
	if ok {
		req.Err <- errors.New(errors.ErrSubscriptionAlreadyExists,
			stderr.New("attempt to create subscription with existing key"))
		return
	}

	m.subs[req.Key] = newSubscription(subscriptionProps{
		Context: m.ctx,
		Logger:  m.logger,
		Key:     req.Key,
		Done:    m.done,
		MQueue:  m.mqueue,
		C:       req.C,
	})

	m.incrSubscriptions()
	m.subs[req.Key].wg.Add(1)
	go m.subs[req.Key].Start()
	req.Err <- nil
}

func (m *SubscriptionManager) destroy(req destroySubscriptionRequest) {
	defer close(req.Err)
	sub, ok := m.subs[req.Key]
	if !ok {
		req.Err <- errors.New(errors.ErrSubscriptionNotFound,
			stderr.New("attempt to destroy subscription that does not exist"))
		return
	}

	sub.Stop()
	m.remove(req.Key)
	m.decrSubscriptions()
	req.Err <- nil
}

func (m *SubscriptionManager) remove(key string) {
	_, ok := m.subs[key]
	if !ok {
		m.logger.Warn(m.ctx, "failed to remove key", log.MapFields{
			"call_type": "RemoveSubscriptionFailure",
			"err":       "key does not exist",
		})
		return
	}

	delete(m.subs, key)
}

// Exists returns true if the subscription exists
func (m *SubscriptionManager) Exists(
	ctx context.Context,
	key string,
) bool {
	out := make(chan bool)
	m.req <- existsSubscriptionRequest{
		Context: ctx,
		Key:     key,
		Out:     out,
	}
	return <-out
}

// Create a new subscription identified by the
// specified key
func (m *SubscriptionManager) Create(
	ctx context.Context,
	key string,
	c chan interface{},
) errors.Err {
	err := make(chan errors.Err)
	m.req <- createSubscriptionRequest{
		Context: ctx,
		Key:     key,
		C:       c,
		Err:     err,
	}
	return <-err
}

// Destroy an existing subscription identified by
// the specified key
func (m *SubscriptionManager) Destroy(
	ctx context.Context,
	key string,
) errors.Err {
	err := make(chan errors.Err)
	m.req <- destroySubscriptionRequest{Context: ctx, Key: key, Err: err}
	return <-err
}

func (m *SubscriptionManager) Stats() stats.Metrics {
	out := make(chan stats.Metrics)
	m.req <- statsRequest{Context: context.Background(), Out: out}
	return <-out
}
