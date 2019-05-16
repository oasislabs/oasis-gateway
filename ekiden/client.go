package ekiden

import (
	"context"
	"errors"

	api "github.com/oasislabs/developer-gateway/ekiden/grpc"
	"google.golang.org/grpc"
)

// Client is an ekiden client instance that keeps hold of a connection
// to an ekiden node
type Client struct {
	conn *grpc.ClientConn
}

// ClientProps are the properties used to initialize and configure
// a client instance on a Dial
type ClientProps struct {
	// URL is the url of the node
	URL string
}

// DialContext dials an ekiden node using the provided ClientProps
func DialContext(ctx context.Context, props *ClientProps) (*Client, error) {
	transport := grpc.WithInsecure()
	conn, err := grpc.DialContext(ctx, props.URL, transport)
	if err != nil {
		return nil, err
	}

	return &Client{conn: conn}, nil
}

// Submit a transaction to the ekiden node and handle the response
func (c *Client) Submit(ctx context.Context, req *SubmitRequest) (*SubmitResponse, error) {
	p, err := MarshalRequest(&SubmitTxRequestPayload{
		Method: req.Method,
		Args:   req.Data,
	})

	if err != nil {
		return nil, err
	}

	runtime := api.NewRuntimeClient(c.conn)
	res, err := runtime.SubmitTx(ctx, &api.SubmitTxRequest{
		RuntimeId: req.RuntimeID,
		Data:      p,
	})
	if err != nil {
		return nil, err
	}

	payload, err := UnmarshalResponse(res.Result)
	if err != nil {
		return nil, err
	}

	if len(payload.Error) > 0 {
		return nil, errors.New(payload.Error)
	}

	return &SubmitResponse{Result: payload}, nil
}
