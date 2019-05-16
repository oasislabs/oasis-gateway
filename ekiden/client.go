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
	p, err := MarshalRequest(&RequestPayload{
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

	var payload ResponsePayload
	if err := UnmarshalResponse(res.Result, &payload); err != nil {
		return nil, err
	}

	if len(payload.Error) > 0 {
		return nil, errors.New(payload.Error)
	}

	return &SubmitResponse{Result: payload}, nil
}

// CallEnclave issues a call to a process running on an enclave
func (c *Client) CallEnclave(ctx context.Context, req *CallEnclaveRequest) (*CallEnclaveResponse, error) {
	p, err := MarshalRequest(&RequestPayload{
		Method: req.Method,
		Args:   req.Data,
	})

	if err != nil {
		return nil, err
	}

	enclave := api.NewEnclaveRpcClient(c.conn)
	res, err := enclave.CallEnclave(ctx, &api.CallEnclaveRequest{
		Endpoint: req.Endpoint,
		Payload:  p,
	})
	if err != nil {
		return nil, err
	}

	var payload ResponsePayload
	if err := UnmarshalResponse(res.Payload, &payload); err != nil {
		return nil, err
	}

	if len(payload.Error) > 0 {
		return nil, errors.New(payload.Error)
	}

	r, ok := payload.Success.([]byte)
	if !ok {
		return nil, errors.New("expected byte array as a response to CallEnclave request")
	}

	return &CallEnclaveResponse{Payload: r}, nil
}

// Submit a transaction to the ekiden node and handle the response
func (c *Client) EthereumTransaction(ctx context.Context, req *EthereumTransactionRequest) (*EthereumTransactionResponse, error) {
	res, err := c.Submit(ctx, &SubmitRequest{
		Method:    "ethereum_transaction",
		RuntimeID: req.RuntimeID,
		Data:      req.Data,
	})
	if err != nil {
		return nil, err
	}

	return &EthereumTransactionResponse{Result: res.Result}, nil
}

// GetPublicKeyRequest retrieves the public key associated with a contract along with
// its metadata
func (c *Client) GetPublicKey(ctx context.Context, req *GetPublicKeyRequest) (*GetPublicKeyResponse, error) {
	res, err := c.CallEnclave(ctx, &CallEnclaveRequest{
		Method: "get_public_key",
		Data:   req.Address[:],
	})
	if err != nil {
		return nil, err
	}

	return &GetPublicKeyResponse{Payload: res.Payload}, nil
}
