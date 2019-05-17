package ekiden

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/tls"
	"errors"
	"fmt"
	"strconv"

	cbor "bitbucket.org/bodhisnarkva/cbor/go"
	api "github.com/oasislabs/developer-gateway/ekiden/grpc"
	"github.com/oasislabs/developer-gateway/noise"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

type FrameSessionID [32]byte

// GenSessionID generates a session ID to talk to the enclave on
// the same connection multiplexing the requests
func GenSessionID(sessionID *FrameSessionID) error {
	n, err := rand.Reader.Read(sessionID[:])
	if err != nil {
		return err
	}

	if n != 32 {
		return errors.New("failed to fill in SessionID random bytes")
	}

	return nil
}

type EnclaveProps struct {
	Endpoint string
	URL      string
}

type Enclave struct {
	conn      *grpc.ClientConn
	session   *noise.Session
	sessionID FrameSessionID
	endpoint  string
}

func DialEnclaveContext(ctx context.Context, props *EnclaveProps) (*Enclave, error) {
	cred := credentials.NewTLS(&tls.Config{InsecureSkipVerify: true})
	transport := grpc.WithTransportCredentials(cred)
	conn, err := grpc.DialContext(ctx, props.URL, transport)
	if err != nil {
		return nil, err
	}

	session, err := noise.NewSession(&noise.SessionProps{
		Initiator: true,
	})
	if err != nil {
		return nil, err
	}

	var sessionID FrameSessionID
	if err := GenSessionID(&sessionID); err != nil {
		return nil, err
	}

	enclave := &Enclave{endpoint: props.Endpoint, conn: conn, session: session, sessionID: sessionID}
	if err = enclave.doHandshake(ctx); err != nil {
		enclave.conn.Close()
		return nil, err
	}

	return enclave, nil
}

func (e *Enclave) doHandshake(ctx context.Context) error {
	enclave := api.NewEnclaveRpcClient(e.conn)
	buf := bytes.NewBuffer(make([]byte, 0, 1024))

	for {
		buf.Reset()
		_, err := e.session.Write(buf, nil)
		if err != nil {
			return err
		}

		requestFrame := Frame{
			SessionID: e.sessionID[:],
			Payload:   buf.Bytes(),
		}

		requestPayload, err := cbor.Dumps(requestFrame)
		if err != nil {
			return err
		}

		res, err := enclave.CallEnclave(ctx, &api.CallEnclaveRequest{
			Endpoint: e.endpoint,
			Payload:  requestPayload,
		}, grpc.WaitForReady(true))
		if err != nil {
			return err
		}

		if e.session.CanUpgrade() {
			break
		}

		buf.Reset()
		if _, err := buf.Write(res.Payload); err != nil {
			return err
		}

		p, _, err := e.session.Read(buf, nil)
		if err != nil {
			return err
		}
		if len(p) > 0 {
			panic("read payload when no request was sent during handshake")
		}

		if e.session.CanUpgrade() {
			break
		}
	}

	session, err := e.session.Upgrade()
	if err != nil {
		return err
	}

	e.session = session
	return nil
}

func (e *Enclave) CallEnclave(ctx context.Context, req *CallEnclaveRequest) (*CallEnclaveResponse, error) {
	p, err := MarshalRequestMessage(&RequestMessage{
		Request: RequestPayload{
			Method: req.Method,
			Args:   req.Data,
		},
	})

	if err != nil {
		return nil, err
	}

	buf := bytes.NewBuffer(make([]byte, 0, 1024))
	_, err = e.session.Write(buf, p)
	if err != nil {
		return nil, err
	}

	requestFrame := Frame{
		SessionID: e.sessionID[:],
		Payload:   buf.Bytes(),
	}

	p, err = MarshalFrame(&requestFrame)
	if err != nil {
		return nil, err
	}

	fmt.Println("BEFORE CALL ENCLAVE: ")
	enclave := api.NewEnclaveRpcClient(e.conn)
	res, err := enclave.CallEnclave(ctx, &api.CallEnclaveRequest{
		Endpoint: e.endpoint,
		Payload:  p,
	})
	fmt.Println("ENCLAVE RESPONSE: ", res, err)

	if err != nil {
		return nil, err
	}

	buf.Reset()
	if _, err := buf.Write(res.Payload); err != nil {
		return nil, err
	}

	p = make([]byte, 0, 1024)
	p, _, err = e.session.Read(buf, p)
	fmt.Println("DECRYPTED PAYLOAD: ", p, err)
	if err != nil {
		return nil, err
	}
	for _, r := range p {
		fmt.Println(strconv.FormatInt(int64(r), 16))
	}

	fmt.Println("DECRYPTED PAYLOAD: ", string(p))
	var payload ResponseMessage
	if err := UnmarshalResponseMessage(p, &payload); err != nil {
		fmt.Println("FAILED TO UNMARSHAL RESPONSE MESSAGE: ", err)
		return nil, err
	}

	fmt.Println("RESPONSE: ", payload)
	if len(payload.Response.Body.Error) > 0 {
		return nil, errors.New(payload.Response.Body.Error)
	}

	return &CallEnclaveResponse{Payload: payload.Response.Body.Success}, nil
}

// GetPublicKeyRequest retrieves the public key associated with a contract along with
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
