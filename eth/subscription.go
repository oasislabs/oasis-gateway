package eth

import (
	"context"
	"fmt"
	"math/big"
	"sync"

	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/oasislabs/developer-gateway/concurrent"
	"github.com/oasislabs/developer-gateway/log"
)

// EthSubscription abstracts an ethereum.Subscription to be
// able to pass a chan<- interface{} and to monitor
// the state of the subscription
type EthSubscription struct {
	sub ethereum.Subscription
	err chan error
}

// Unsubscribe destroys the subscription
func (s *EthSubscription) Unsubscribe() {
	s.sub.Unsubscribe()
}

// Err returns a channel to retrieve subscription errors.
// Only one error at most will be sent through this chanel,
// when the subscription is closed, this channel will be closed
// so this can be used by a client to monitor whether the
// subscription is active
func (s *EthSubscription) Err() <-chan error {
	return s.err
}

// LogSubscriber creates log based subscriptions
// using the underlying clients
type LogSubscriber struct {
	lock        sync.Mutex
	FilterQuery ethereum.FilterQuery
	BlockNumber uint64
	Index       uint
}

func (s *LogSubscriber) createSubscription(
	ctx context.Context,
	client Client,
	clog chan<- types.Log,
) (ethereum.Subscription, error) {
	s.lock.Lock()
	defer s.lock.Unlock()
	return client.SubscribeFilterLogs(ctx, s.FilterQuery, clog)
}

// Subscribe implementation of Subscriber for LogSubscriber
func (s *LogSubscriber) Subscribe(
	ctx context.Context,
	client Client,
	c chan<- interface{},
) (ethereum.Subscription, error) {
	cerr := make(chan error)
	clog := make(chan types.Log, 64)

	sub, err := s.createSubscription(ctx, client, clog)
	if err != nil {
		return nil, err
	}

	go func() {
		defer func() {
			// ensure that if the subscriber is started again it will start
			// from the block from which it stopped
			s.lock.Lock()
			defer s.lock.Unlock()
			s.FilterQuery.FromBlock = big.NewInt(0).SetUint64(s.BlockNumber)
			close(cerr)
		}()

		for {
			select {
			case <-ctx.Done():
				return
			case ev, ok := <-clog:
				if !ok {
					return
				}

				// in case events are received that are previous to the offsets
				// tracked by the subscriber, the events are discarded
				if ev.BlockNumber < s.BlockNumber ||
					(ev.BlockNumber == s.BlockNumber && ev.Index < s.Index) {
					continue
				}

				s.BlockNumber = ev.BlockNumber
				s.Index = ev.Index

				c <- ev
			case err, ok := <-sub.Err():
				if !ok {
					return
				}

				cerr <- err
				return
			}
		}
	}()

	return &EthSubscription{sub: sub, err: cerr}, nil
}

// Subscriber is an interface for types that creates subscriptions
// against an ethereum-like backend
type Subscriber interface {
	// Subscribe creates a subscription and forwards the received
	// events on the provided channel
	Subscribe(context.Context, Client, chan<- interface{}) (ethereum.Subscription, error)
}

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

	// C channel to receive the events for a subscription
	C chan<- interface{}
}

// Subscription abstracts an ethereum subscription into a type
// that implements automatic dialing and retries
type Subscription struct {
	logger     log.Logger
	client     Client
	sub        ethereum.Subscription
	subscriber Subscriber
	url        string
	key        string
	c          chan<- interface{}
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
	if props.Subscriber == nil {
		panic("subscriber must be set")
	}
	if props.C == nil {
		panic("receiving channel must be set")
	}

	s := &Subscription{
		logger:     props.Logger.ForClass("eth", "Subscription"),
		client:     props.Client,
		url:        props.URL,
		subscriber: props.Subscriber,
		key:        props.Key,
		c:          props.C,
	}

	return s
}

func (s *Subscription) subscribe(ctx context.Context) error {
	sub, err := s.subscriber.Subscribe(ctx, s.client, s.c)
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

func (s *Subscription) Unsubscribe() {
	s.sub.Unsubscribe()
}

func (s *Subscription) handle(ctx context.Context, ev concurrent.WorkerEvent) (interface{}, error) {
	switch ev := ev.(type) {
	case concurrent.RequestWorkerEvent:
		panic("no requests should be issued to the subscription")
	case concurrent.ErrorWorkerEvent:
		err := s.handleError(ctx, ev)
		return nil, err
	default:
		panic("received unexpected event type")
	}
}

func (s *Subscription) handleError(ctx context.Context, ev concurrent.ErrorWorkerEvent) error {
	s.logger.Debug(ctx, "subscription failed, recreating", log.MapFields{
		"call_type": "CurrentSubscriptionFailure",
		"err":       ev.Error.Error(),
	})

	return s.subscribe(ctx)
}

type createSubscriptionRequest struct {
	C          chan<- interface{}
	Subscriber Subscriber
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
	client Client
	master *concurrent.Master
}

// NewSubscriptionManager creates a new subscription manager
func NewSubscriptionManager(
	props SubscriptionManagerProps,
) *SubscriptionManager {
	m := SubscriptionManager{
		ctx:    props.Context,
		logger: props.Logger.ForClass("eth", "SubscriptionManager"),
		client: props.Client,
	}

	m.master = concurrent.NewMaster(concurrent.MasterProps{
		MasterHandler: concurrent.MasterHandlerFunc(m.handle),
	})

	if err := m.master.Start(props.Context); err != nil {
		panic(fmt.Sprintf("failed to start loop %s", err.Error()))
	}

	return &m
}

func (m *SubscriptionManager) handle(ctx context.Context, ev concurrent.MasterEvent) error {
	switch ev := ev.(type) {
	case concurrent.CreateWorkerEvent:
		return m.create(ctx, ev)
	case concurrent.DestroyWorkerEvent:
		return m.destroy(ev)
	default:
		panic("received unknown request")
	}
}

func (m *SubscriptionManager) create(ctx context.Context, ev concurrent.CreateWorkerEvent) error {
	req := ev.Value.(createSubscriptionRequest)
	sub := NewSubscription(SubscriptionProps{
		Logger:     m.logger,
		Client:     m.client,
		Key:        ev.Key,
		Subscriber: req.Subscriber,
		C:          req.C,
	})

	// inherit context from manager so that cancelling the manager's context
	// will cancel all subscriptions
	if err := sub.subscribe(m.ctx); err != nil {
		return err
	}

	ev.Props.ErrC = sub.sub.Err()
	ev.Props.WorkerHandler = concurrent.WorkerHandlerFunc(sub.handle)
	ev.Props.UserData = sub

	return nil
}

func (m *SubscriptionManager) destroy(ev concurrent.DestroyWorkerEvent) error {
	sub := ev.Worker.UserData.(*Subscription)
	sub.Unsubscribe()
	return nil
}

// Create a new subscription identified by the
// specified key
func (m *SubscriptionManager) Create(
	ctx context.Context,
	key string,
	subscriber Subscriber,
	c chan<- interface{},
) error {
	if len(key) == 0 {
		panic("key must be set")
	}

	if subscriber == nil {
		panic("subscriber must not be nil")
	}

	return m.master.Create(ctx, key, createSubscriptionRequest{
		Subscriber: subscriber,
		C:          c,
	})
}

// Destroy an existing subscription identified by
// the specified key
func (m *SubscriptionManager) Destroy(
	ctx context.Context,
	key string,
) error {
	return m.master.Destroy(ctx, key)
}
