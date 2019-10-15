package ekiden

import (
	"bytes"
	"context"
	"crypto/tls"
	"errors"
	"io"

	api "github.com/oasislabs/oasis-gateway/ekiden/grpc"
	"github.com/oasislabs/oasis-gateway/noise"
	"github.com/oasislabs/oasis-gateway/rw"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

type EnclaveProps struct {
	Endpoint string
	URL      string
}

type Enclave struct {
	conn     *grpc.ClientConn
	client   *noise.Client
	endpoint string
}

func DialEnclaveContext(ctx context.Context, props *EnclaveProps) (*Enclave, error) {
	cred := credentials.NewTLS(&tls.Config{InsecureSkipVerify: true})
	transport := grpc.WithTransportCredentials(cred)
	conn, err := grpc.DialContext(ctx, props.URL, transport)
	if err != nil {
		return nil, err
	}

	enclave := &Enclave{endpoint: props.Endpoint, conn: conn}

	client, err := noise.DialContext(ctx, noise.ClientProps{
		Conns:  1,
		Client: noise.ClientFunc(enclave.request),
		SessionProps: noise.SessionProps{
			Initiator: true,
		},
	})
	if err != nil {
		return nil, err
	}

	enclave.client = client
	return enclave, nil
}

// request is used as the underlying channel to communicate with the
// enclave.
func (e *Enclave) request(ctx context.Context, w io.Writer, r io.Reader) error {
	buf := bytes.NewBuffer(make([]byte, 0, 128))
	enclave := api.NewEnclaveRpcClient(e.conn)

	if _, err := rw.CopyWithLimit(buf, r, rw.ReadLimitProps{
		FailOnExceed: true,
		Limit:        65535,
	}); err != nil {
		return err
	}

	res, err := enclave.CallEnclave(ctx, &api.CallEnclaveRequest{
		Endpoint: e.endpoint,
		Payload:  buf.Bytes(),
	})
	if err != nil {
		return err
	}

	if _, err := rw.CopyWithLimit(w, bytes.NewReader(res.Payload), rw.ReadLimitProps{
		FailOnExceed: true,
		Limit:        65535,
	}); err != nil {
		return err
	}

	return nil
}

func (e *Enclave) CallEnclave(ctx context.Context, req *CallEnclaveRequest) (*CallEnclaveResponse, error) {
	res, err := e.client.Request(ctx, noise.RequestPayload{
		Method: req.Method,
		Args:   req.Data,
	})

	if err != nil {
		return nil, err
	}

	if len(res.Error) > 0 {
		return nil, errors.New(res.Error)
	}

	return &CallEnclaveResponse{Payload: res.Success}, nil
}

// GetCode retrieves the code associated with a service along with
// its metadata
func (e *Enclave) GetCode(ctx context.Context, req *GetCodeRequest) (*GetCodeResponse, error) {
	res, err := e.CallEnclave(ctx, &CallEnclaveRequest{
		Method: "get_code",
		Data:   req.Address[:],
	})
	if err != nil {
		return nil, err
	}

	if res.Payload == nil {
		return nil, errors.New("Provided address does not have associated source code")
	}

	return nil, errors.New("GetCode not fully implemented")
}

// GetPublicKeyRequest retrieves the public key associated with a service along with
// its metadata
func (e *Enclave) GetPublicKey(ctx context.Context, req *GetPublicKeyRequest) (*GetPublicKeyResponse, error) {
	res, err := e.CallEnclave(ctx, &CallEnclaveRequest{
		Method: "get_public_key",
		Data:   req.Address[:],
	})
	if err != nil {
		return nil, err
	}

	if res.Payload == nil {
		return nil, errors.New("Provided address does not have an associated public key")
	}

	return nil, errors.New("GetPublicKey not fully implemented")
}
