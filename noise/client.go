package noise

import (
	"context"
)

type response struct {
	Error    error
	Response ResponsePayload
}

type request struct {
	Context  context.Context
	Request  RequestPayload
	Response chan response
}

// Client manages a fixed pool of connections and distributes work amongst
// them so that the caller does not need to worry about concurrency
type Client struct {
	c chan request
}

// ClientProps sets up the connection pool
type ClientProps struct {
	Conns        int
	Client       Requester
	SessionProps SessionProps
}

// DialContext creates a new pool of connections
func DialContext(ctx context.Context, props ClientProps) (*Client, error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	pool := &Client{c: make(chan request, 64)}

	for i := 0; i < props.Conns; i++ {
		// TODO(stan): this can be done in parallel
		if err := pool.dialConnection(ctx, props.Client, &props.SessionProps); err != nil {
			return nil, err
		}
	}

	return pool, nil
}

// Request issues a request to one of the connections in the pool and
// retrieves the response. The pool is concurrency safe.
func (p *Client) Request(ctx context.Context, req RequestPayload) (ResponsePayload, error) {
	res := make(chan response)
	p.c <- request{Context: ctx, Request: req, Response: res}
	response := <-res
	return response.Response, response.Error
}

func startConnLoop(ctx context.Context, conn *Conn, c <-chan request) {
	for {
		select {
		case <-ctx.Done():
			return
		case req, ok := <-c:
			if !ok {
				return
			}

			res, err := conn.Request(req.Context, req.Request)
			req.Response <- response{Error: err, Response: res}
		}
	}
}

func (p *Client) dialConnection(ctx context.Context, client Requester, props *SessionProps) error {
	conn, err := DialConnContext(ctx, client, props)
	if err != nil {
		// TODO(stan): if a connection fails to establish we should shutdown
		// all the successful connection gracefully
		return err
	}

	go startConnLoop(ctx, conn, p.c)
	return nil
}
