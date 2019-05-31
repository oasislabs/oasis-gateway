package eth

import (
	"context"
	"strings"

	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	rpc "github.com/ethereum/go-ethereum/rpc"
	"github.com/oasislabs/developer-gateway/conc"
)

type Client interface {
	EstimateGas(context.Context, ethereum.CallMsg) (uint64, error)
	GetPublicKey(context.Context, common.Address) (PublicKey, error)
	PendingNonceAt(context.Context, common.Address) (uint64, error)
	SendTransaction(context.Context, *types.Transaction) error
	SubscribeFilterLogs(context.Context, ethereum.FilterQuery, chan<- types.Log) (ethereum.Subscription, error)
	TransactionReceipt(ctx context.Context, txHash common.Hash) (*types.Receipt, error)
}

type Pool interface {
	Conn(context.Context) (*Conn, error)
	Report(context.Context, *Conn) error
}

type PooledClientProps struct {
	Pool        Pool
	RetryConfig conc.RetryConfig
}

func NewPooledClient(props PooledClientProps) *PooledClient {
	return &PooledClient{
		pool:        props.Pool,
		retryConfig: props.RetryConfig,
	}
}

type PooledClient struct {
	pool        Pool
	retryConfig conc.RetryConfig
}

func (c *PooledClient) shouldRetryAfterError(err error) bool {
	// TODO(stan): find out what's the right condition for returning
	// a client to the pool in case of failure

	switch {
	case strings.Contains(err.Error(), "Requested gas greater than block gas limit"):
		return false
	case strings.Contains(err.Error(), "Invalid transaction nonce"):
		return false
	default:
		return true
	}
}

func (c *PooledClient) request(ctx context.Context, fn func(conn *Conn) (interface{}, error)) (interface{}, error) {
	return conc.RetryWithConfig(ctx, conc.SupplierFunc(func() (interface{}, error) {
		conn, err := c.pool.Conn(ctx)
		if err != nil {
			return nil, err
		}

		v, err := fn(conn)
		if err != nil {
			if c.shouldRetryAfterError(err) {
				return nil, err

			} else {
				return nil, conc.ErrCannotRecover{Cause: err}
			}
		}

		return v, nil
	}), c.retryConfig)
}

func (c *PooledClient) EstimateGas(ctx context.Context, msg ethereum.CallMsg) (uint64, error) {
	v, err := c.request(ctx, func(conn *Conn) (interface{}, error) {
		return conn.eclient.EstimateGas(ctx, msg)
	})

	if err != nil {
		return 0, err
	}

	return v.(uint64), nil
}

func (c *PooledClient) GetPublicKey(ctx context.Context, address common.Address) (PublicKey, error) {
	v, err := c.request(ctx, func(conn *Conn) (interface{}, error) {
		var pk PublicKey
		err := conn.rclient.CallContext(ctx, &pk, "oasis_getPublicKey", address)
		return pk, err
	})

	if err != nil {
		return PublicKey{}, err
	}

	return v.(PublicKey), nil
}

func (c *PooledClient) PendingNonceAt(ctx context.Context, account common.Address) (uint64, error) {
	v, err := c.request(ctx, func(conn *Conn) (interface{}, error) {
		return conn.eclient.PendingNonceAt(ctx, account)
	})

	if err != nil {
		return 0, err
	}

	return v.(uint64), nil
}

func (c *PooledClient) SendTransaction(ctx context.Context, tx *types.Transaction) error {
	_, err := c.request(ctx, func(conn *Conn) (interface{}, error) {
		err := conn.eclient.SendTransaction(ctx, tx)
		return nil, err
	})

	return err
}

func (c *PooledClient) TransactionReceipt(ctx context.Context, txHash common.Hash) (*types.Receipt, error) {
	v, err := c.request(ctx, func(conn *Conn) (interface{}, error) {
		return conn.eclient.TransactionReceipt(ctx, txHash)
	})

	if err != nil {
		return nil, err
	}

	return v.(*types.Receipt), nil
}

func (c *PooledClient) SubscribeFilterLogs(
	ctx context.Context,
	q ethereum.FilterQuery,
	ch chan<- types.Log,
) (ethereum.Subscription, error) {
	v, err := c.request(ctx, func(conn *Conn) (interface{}, error) {
		return conn.eclient.SubscribeFilterLogs(ctx, q, ch)
	})

	if err != nil {
		return nil, err
	}

	return v.(ethereum.Subscription), nil
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

type UniDialerProps struct {
	URL         string
	RetryConfig conc.RetryConfig
}

// UniDialer implements the Dialer interface and it provides
// a connection to a specific URL. If a different URL is attempted
// the FixedDialer will return an error
type UniDialer struct {
	ctx  context.Context
	conn *Conn
	url  string
	req  chan interface{}
}

// NewUniDialer keeps a connection open to an endpoint. If the
// connection needs to be recreated a client can signal the pool
// to recreate the connection. Only websocket endpoints are
// supported because only websocket endpoints support
// the subscribe API
func NewUniDialer(ctx context.Context, url string) *UniDialer {
	p := UniDialer{ctx: ctx, conn: nil, url: url, req: make(chan interface{})}
	go p.startLoop()
	return &p
}

func (p *UniDialer) startLoop() {
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

func (p *UniDialer) request(req interface{}) {
	switch req := req.(type) {
	case dialRequest:
		p.dial(req)
	case returnRequest:
		p.returnClient(req)
	default:
		panic("received unknown request object")
	}
}

func (p *UniDialer) returnClient(req returnRequest) {
	if p.conn == req.Conn {
		p.conn = nil
	}

	req.C <- returnResponse{Error: nil}
}

func (p *UniDialer) dial(req dialRequest) {
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

// Report returns a failed Client connection. In this
// case, the pool we create a new Client connection on the
// next DialContext
func (p *UniDialer) Report(ctx context.Context, conn *Conn) error {
	c := make(chan returnResponse)
	p.req <- returnRequest{C: c, Conn: conn}
	res := <-c
	return res.Error
}

// DialContext implementation of Dialer for FixedDialer
func (p *UniDialer) Conn(ctx context.Context) (*Conn, error) {
	c := make(chan dialResponse)
	p.req <- dialRequest{Context: ctx, C: c}
	res := <-c
	return res.Conn, res.Error
}
