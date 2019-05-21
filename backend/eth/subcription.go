package eth

import (
	"context"

	"github.com/oasislabs/developer-gateway/errors"
	"github.com/oasislabs/developer-gateway/eth"
	"github.com/oasislabs/developer-gateway/log"
)

type createSubscriptionRequest struct {
	Key string
	Err chan<- error
}

type destroySubscriptionRequest struct {
	Key string
	Err chan<- error
}

type SubscriptionManagerProps struct {
	Logger log.Logger
}

// SubscriptionManager manages the lifetime
// of a group of subscriptions
type SubscriptionManager struct {
	ctx    context.Context
	logger log.Logger
	done   chan eth.SubscriptionEndEvent
	req    chan interface{}
	subs   map[string]*eth.Subscription
}

// NewSubscriptionManager creates a new subscription manager
func NewSubscriptionManager(ctx context.Context, logger log.Logger) *SubscriptionManager {
	m := SubscriptionManager{
		ctx:    ctx,
		logger: logger.ForClass("eth", "SubscriptionManager"),
		done:   make(chan eth.SubscriptionEndEvent),
		req:    make(chan interface{}),
		subs:   make(map[string]eth.Subscription),
	}

	go m.startLoop()
	return &m
}

func (m *SubscriptionManager) startLoop() {
	defer func() {
		for _, sub := range m.subs {
			sub.Unsubscribe()
			m.remove(sub.Key())
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
	default:
		panic("received unknown request")
	}
}

func (m *SubscriptionManager) create(req createSubscriptionRequest) {
	_, ok := m.subs[req.Key]
	if ok {
		req.Err <- errors.New("subscription already exists")
		return
	}

	m.subs[req.Key] = eth.NewSubscription(eth.SubscriptionProps{})

	req.Err <- nil
}

func (m *SubscriptionManager) destroy(req destroySubscriptionRequest) {
	sub, ok := m.subs[req.Key]
	if !ok {
		req.Err <- errors.New("subscription does not exist")
		return
	}

	sub.Unsubscribe()
	m.remove(req.Key)
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

// Create a new subscription identified by the
// specified key
func (m *SubscriptionManager) Create(key string) error {
	err := make(chan error)
	m.req <- createSubscriptionRequest{Key: key, Err: err}
	return <-err
}

// Destroy an existing subscription identified by
// the specified key
func (m *SubscriptionManager) Destroy(key string) error {
	err := make(chan error)
	m.req <- destroySubscriptionRequest{Key: key, Err: err}
	return <-err
}
