package eth

import (
	"context"
	"errors"

	ethereum "github.com/ethereum/go-ethereum"
	"github.com/oasislabs/developer-gateway/log"
)

// SubscriptionEndEvent is issued to the subscription supervisor
// when the subscription has been destroyed
type SubscriptionEndEvent struct {
	// Key uniquely identifies a subscription. It is provided by the
	// supervisor
	Key string

	// Error in case the subscription ended because of an error
	Error error
}

// SubscriptionProps are the properties required when
// creating a subscription
type SubscriptionProps struct {
	// Context used by the subscription and that can be used
	// to signal a cancellation of the subcription
	Context context.Context

	// Logger used by the subscription
	Logger log.Logger

	// Client to make requests
	Client Client

	// URL to dial to for the subscription. Must be a ws URL since
	// other provides may not support creating subscriptions
	URL string

	// Key uniquely identifies a subscription and it can be used to
	// notify the subscription supervisor
	Key string

	// Subscriber used to create the subscription
	Subscriber Subscriber

	// C is the channel used by the subscription to send the received
	// events
	C chan<- interface{}

	// Done is called by the subscription when a subscription exits
	Done chan<- SubscriptionEndEvent
}

// Subscription abstracts an ethereum subscription into a type
// that implements automatic dialing and retries
type Subscription struct {
	ctx        context.Context
	cancel     context.CancelFunc
	logger     log.Logger
	client     Client
	sub        ethereum.Subscription
	subscriber Subscriber
	url        string
	key        string
	done       chan<- SubscriptionEndEvent
	c          chan<- interface{}
	consumer   chan interface{}
}

// NewSubscription creates a new subscription with the
// passed properties
func NewSubscription(props SubscriptionProps) *Subscription {
	if props.Logger == nil {
		panic("logger cannot be nil")
	}
	if props.Client == nil {
		panic("client must be set")
	}
	if props.C == nil {
		panic("channel must be set")
	}
	if props.Subscriber == nil {
		panic("subscriber must be set")
	}

	if props.Context == nil {
		props.Context = context.Background()
	}

	ctx, cancel := context.WithCancel(props.Context)
	s := &Subscription{
		ctx:        ctx,
		cancel:     cancel,
		logger:     props.Logger.ForClass("eth", "Subscription"),
		client:     props.Client,
		url:        props.URL,
		subscriber: props.Subscriber,
		key:        props.Key,
		done:       props.Done,
		c:          props.C,
		consumer:   make(chan interface{}, 64),
	}

	go s.startLoop()

	return s
}

func (s *Subscription) subscribe() error {
	sub, err := s.subscriber.Subscribe(s.ctx, s.client, s.c)
	if err != nil {
		return err
	}

	s.sub = sub.(ethereum.Subscription)
	return nil
}

// Key uniquely identifies the subscription within the global
// space of subscriptions
func (s *Subscription) Key() string {
	return s.key
}

// Unsubscribe stops the subscription
func (s *Subscription) Unsubscribe() {
	s.cancel()
}

func (s *Subscription) startLoop() {
	defer func() {
		close(s.done)
	}()

	if err := s.subscribe(); err != nil {
		s.done <- SubscriptionEndEvent{Key: s.key, Error: err}
		return
	}

	for {
		select {
		case <-s.ctx.Done():
			s.done <- SubscriptionEndEvent{Key: s.key, Error: nil}
			return
		case err, ok := <-s.sub.Err():
			if ok {
				s.logger.Debug(s.ctx, "subscription failed, recreating", log.MapFields{
					"call_type": "CurrentSubscriptionFailure",
					"err":       err.Error(),
				})
			}

			if err := s.subscribe(); err != nil {
				s.done <- SubscriptionEndEvent{Key: s.key, Error: err}
				return
			}
		}
	}
}

type createSubscriptionRequest struct {
	Context    context.Context
	Key        string
	Err        chan<- error
	C          chan<- interface{}
	Subscriber Subscriber
}

type destroySubscriptionRequest struct {
	Context context.Context
	Key     string
	Err     chan<- error
}

// SubscriptionManagerProps properties used to create the
// behaviour of the manager and the subscriptions created
type SubscriptionManagerProps struct {
	// Context used by the manager and that can be used
	// to signal a cancellation
	Context context.Context

	// Logger used by the manager and its subscriptions
	Logger log.Logger

	// Client to make requests
	Client Client
}

// SubscriptionManager manages the lifetime
// of a group of subscriptions
type SubscriptionManager struct {
	ctx    context.Context
	logger log.Logger
	done   chan SubscriptionEndEvent
	req    chan interface{}
	subs   map[string]*Subscription
	client Client
}

// NewSubscriptionManager creates a new subscription manager
func NewSubscriptionManager(props SubscriptionManagerProps) *SubscriptionManager {
	m := SubscriptionManager{
		ctx:    props.Context,
		logger: props.Logger.ForClass("eth", "SubscriptionManager"),
		done:   make(chan SubscriptionEndEvent),
		req:    make(chan interface{}),
		subs:   make(map[string]*Subscription),
		client: props.Client,
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
		m.destroy(req)
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

	m.subs[req.Key] = NewSubscription(SubscriptionProps{
		Context:    m.ctx,
		Logger:     m.logger,
		Client:     m.client,
		Key:        req.Key,
		Subscriber: req.Subscriber,
		C:          req.C,
		Done:       m.done,
	})

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
func (m *SubscriptionManager) Create(
	ctx context.Context,
	key string,
	subscriber Subscriber,
	c chan<- interface{},
) error {
	err := make(chan error)
	m.req <- createSubscriptionRequest{
		Context:    ctx,
		Key:        key,
		Subscriber: subscriber,
		C:          c,
		Err:        err,
	}
	return <-err
}

// Destroy an existing subscription identified by
// the specified key
func (m *SubscriptionManager) Destroy(
	ctx context.Context,
	key string,
) error {
	err := make(chan error)
	m.req <- destroySubscriptionRequest{Context: ctx, Key: key, Err: err}
	return <-err
}
