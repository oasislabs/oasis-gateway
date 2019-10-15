package ekiden

import (
	"context"
	"errors"

	api "github.com/oasislabs/oasis-gateway/ekiden/grpc"
	"google.golang.org/grpc"
)

type Runtime struct {
	conn *grpc.ClientConn
}

func DialRuntimeContext(ctx context.Context, url string) (*Runtime, error) {
	transport := grpc.WithInsecure()
	conn, err := grpc.DialContext(ctx, url, transport)
	if err != nil {
		return nil, err
	}

	return &Runtime{conn: conn}, nil
}

// Submit a transaction to the ekiden node and handle the response
func (r *Runtime) Submit(ctx context.Context, req *SubmitRequest) (*SubmitResponse, error) {
	p, err := MarshalRequest(&RequestPayload{
		Method: req.Method,
		Args:   req.Data,
	})

	if err != nil {
		return nil, err
	}

	runtime := api.NewRuntimeClient(r.conn)
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

// Submit a transaction to the ekiden node and handle the response
func (r *Runtime) EthereumTransaction(
	ctx context.Context,
	req *EthereumTransactionRequest,
) (*EthereumTransactionResponse, error) {
	res, err := r.Submit(ctx, &SubmitRequest{
		Method:    "ethereum_transaction",
		RuntimeID: req.RuntimeID,
		Data:      req.Data,
	})
	if err != nil {
		return nil, err
	}

	return &EthereumTransactionResponse{Result: res.Result}, nil
}
