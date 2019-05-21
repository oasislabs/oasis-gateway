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

// FixedConnPool manages a fixed pool of connections and distributes work amongst
// them so that the caller does not need to worry about concurrency
type FixedConnPool struct {
	c chan request
}

// FixedConnPoolProps sets up the connection pool
type FixedConnPoolProps struct {
	Conns        int
	Client       Client
	SessionProps SessionProps
}

// DialFixedPool creates a new pool of connections
func DialFixedPool(ctx context.Context, props FixedConnPoolProps) (*FixedConnPool, error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	pool := &FixedConnPool{c: make(chan request, 64)}

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
func (p *FixedConnPool) Request(ctx context.Context, req RequestPayload) (ResponsePayload, error) {
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

func (p *FixedConnPool) dialConnection(ctx context.Context, client Client, props *SessionProps) error {
	conn, err := DialContext(ctx, client, props)
	if err != nil {
		// TODO(stan): if a connection fails to establish we should shutdown
		// all the successful connection gracefully
		return err
	}

	go startConnLoop(ctx, conn, p.c)
	return nil
}
