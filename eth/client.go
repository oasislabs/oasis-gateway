package eth

import (
	"context"
	"math/big"

	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	rpc "github.com/ethereum/go-ethereum/rpc"
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
	FilterQuery ethereum.FilterQuery
	BlockNumber uint64
	Index       uint
}

// Subscribe implementation of Subscriber for LogSubscriber
func (s *LogSubscriber) Subscribe(
	ctx context.Context,
	client Client,
	c chan<- interface{},
) (ethereum.Subscription, error) {
	clog := make(chan types.Log, 64)
	cerr := make(chan error)

	sub, err := client.SubscribeFilterLogs(ctx, s.FilterQuery, clog)
	if err != nil {
		return nil, err
	}

	go func() {
		defer func() {
			// ensure that if the subscriber is started again it will start
			// from the block from which it stopped
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
					(ev.BlockNumber == s.BlockNumber && ev.Index <= s.Index) {
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

type Client interface {
	SubscribeFilterLogs(context.Context, ethereum.FilterQuery, chan<- types.Log) (ethereum.Subscription, error)
}

type PooledClient struct {
	pool *FixedPoolDialer
}

func (c *PooledClient) SubscribeFilterLogs(
	ctx context.Context,
	q ethereum.FilterQuery,
	ch chan<- types.Log,
) (ethereum.Subscription, error) {
	conn, err := c.pool.Conn(ctx)
	if err != nil {
		return nil, err
	}

	return conn.eclient.SubscribeFilterLogs(ctx, q, ch)
}

type Conn struct {
	eclient *ethclient.Client
	rclient *rpc.Client
}

type dialResponse struct {
	Conn  *Conn
	Error error
}

type dialRequest struct {
	Context context.Context
	C       chan<- dialResponse
}

type returnRequest struct {
	Conn *Conn
	C    chan<- returnResponse
}

type returnResponse struct {
	Error error
}

// FixedPoolDialer implements the Dialer interface and it provides
// a connection to a specific URL. If a different URL is attempted
// the FixedDialer will return an error
type FixedPoolDialer struct {
	ctx  context.Context
	conn *Conn
	url  string
	req  chan interface{}
}

// NewFixedPoolDialer a connection open to an endpoint. If the
// connection needs to be recreated a client can signal the pool
// to recreate the connection. Only websocket endpoints are
// supported because only websocket endpoints support
// the subscribe API
func NewFixedPoolDialer(ctx context.Context, url string) *FixedPoolDialer {
	p := FixedPoolDialer{ctx: ctx, conn: nil, url: url, req: make(chan interface{})}
	go p.startLoop()
	return &p
}

func (p *FixedPoolDialer) startLoop() {
	defer func() {
		p.conn.rclient.Close()
	}()

	for {
		select {
		case <-p.ctx.Done():
			return
		case req := <-p.req:
			p.request(req)
		}
	}
}

func (p *FixedPoolDialer) request(req interface{}) {
	switch req := req.(type) {
	case dialRequest:
		p.dial(req)
	case returnRequest:
		p.returnClient(req)
	default:
		panic("received unknown request object")
	}
}

func (p *FixedPoolDialer) returnClient(req returnRequest) {
	if p.conn == req.Conn {
		p.conn = nil
	}

	req.C <- returnResponse{Error: nil}
}

func (p *FixedPoolDialer) dial(req dialRequest) {
	if p.conn != nil {
		req.C <- dialResponse{Conn: p.conn, Error: nil}
		return
	}

	c, err := rpc.DialWebsocket(req.Context, p.url, "")
	if err != nil {
		req.C <- dialResponse{Conn: nil, Error: err}
		return
	}

	p.conn = &Conn{
		eclient: ethclient.NewClient(c),
		rclient: c,
	}

	req.C <- dialResponse{Conn: p.conn, Error: nil}
}

// ReturnFailed returns a failed Client connection. In this
// case, the pool we create a new Client connection on the
// next DialContext
func (p *FixedPoolDialer) ReturnFailed(ctx context.Context, conn *Conn) error {
	c := make(chan returnResponse)
	p.req <- returnRequest{C: c, Conn: conn}
	res := <-c
	return res.Error
}

// DialContext implementation of Dialer for FixedDialer
func (p *FixedPoolDialer) Conn(ctx context.Context) (*Conn, error) {
	c := make(chan dialResponse)
	p.req <- dialRequest{Context: ctx, C: c}
	res := <-c
	return res.Conn, res.Error
}
