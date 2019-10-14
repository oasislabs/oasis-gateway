package service

import (
	"context"
	"encoding/binary"
	"encoding/hex"
	stderr "errors"

	auth "github.com/oasislabs/developer-gateway/auth/core"
	backend "github.com/oasislabs/developer-gateway/backend/core"
	"github.com/oasislabs/developer-gateway/errors"
	"github.com/oasislabs/developer-gateway/log"
	"github.com/oasislabs/developer-gateway/rpc"
)

// Client interface for the underlying operations needed for the API
// implementation
type Client interface {
	// DeployServiceAsync triggers a deploy service operation and returns an ID with which
	// the response can be later retrieved with a PollService request
	DeployServiceAsync(context.Context, backend.DeployServiceRequest) (uint64, errors.Err)

	// ExecuteServiceAsync triggers an execute service operation and returns an ID with which
	// the response can be later retrieved with a PollService request
	ExecuteServiceAsync(context.Context, backend.ExecuteServiceRequest) (uint64, errors.Err)

	// PollService allows the client to poll for asynchronous responses
	PollService(context.Context, backend.PollServiceRequest) (backend.Events, errors.Err)

	// GetCode retrieves the code associated with a service.
	GetCode(context.Context, backend.GetCodeRequest) (backend.GetCodeResponse, errors.Err)

	// GetPublicKey retrieves the public key associated with a service
	// so that the client can encrypt and format the input data in a confidential
	// and privacy preserving manner.
	GetPublicKey(context.Context, backend.GetPublicKeyRequest) (backend.GetPublicKeyResponse, errors.Err)
}

// Services required by the ServiceHandler execution
type Services struct {
	Logger   log.Logger
	Client   Client
	Verifier auth.Auth
}

// ServiceHandler implements the handlers for service management
type ServiceHandler struct {
	logger   log.Logger
	client   Client
	verifier auth.Auth
}

// DeployService handles the deployment of new services
func (h ServiceHandler) DeployService(ctx context.Context, v interface{}) (interface{}, error) {
	aad := ctx.Value(auth.AAD{}).(string)
	session := ctx.Value(auth.Session{}).(string)
	req := v.(*DeployServiceRequest)

	authReq := auth.AuthRequest{
		API:  "Deploy",
		Data: req.Data,
	}

	if err := h.verifier.Verify(ctx, authReq); err != nil {
		e := errors.New(errors.ErrFailedAADVerification, err)
		h.logger.Debug(ctx, "failed to verify AAD", log.MapFields{
			"call_type": "DeployServiceFailure",
			"session":   session,
			"err":       e,
		})
		return nil, e
	}

	// a context from an http request is cancelled after the response to the request is returned,
	// so a new context is needed to handle the asynchronous request
	id, err := h.client.DeployServiceAsync(context.Background(), backend.DeployServiceRequest{
		AAD:        aad,
		Data:       req.Data,
		SessionKey: session,
	})
	if err != nil {
		h.logger.Debug(ctx, "failed to start request", log.MapFields{
			"call_type": "DeployServiceFailure",
			"session":   session,
		}, err)
		return nil, err
	}

	return AsyncResponse{ID: id}, nil
}

// parseExecuteMessage attempts to extract the AAD and PK from a standard confidential message format.
func (h ServiceHandler) parseExecuteMessage(v *ExecuteServiceRequest) (authReq auth.AuthRequest) {
	authReq.API = "Execute"
	authReq.Address = v.Address
	authReq.Data = v.Data

	// Attempt to parse data as hex-encoded bytes.
	dst := make([]byte, hex.DecodedLen(len(v.Data)))
	_, err := hex.Decode(dst, []byte(v.Data))
	if err != nil || len(dst) < 32 {
		return
	}

	// PK is 1st 16 bytes.
	authReq.PK = dst[0:16]
	// extract msg and aad length after PK
	cipherLength := binary.BigEndian.Uint64([]byte(dst[16:24]))
	aadLength := binary.BigEndian.Uint64([]byte(dst[24:32]))

	if len(dst) < int(cipherLength+aadLength+32) {
		return
	}

	// AAD slice.
	aadOffset := 32 + cipherLength
	authReq.AAD = dst[aadOffset : aadOffset+aadLength]

	return
}

// ExecuteService handle the execution of deployed services
func (h ServiceHandler) ExecuteService(ctx context.Context, v interface{}) (interface{}, error) {
	aad := ctx.Value(auth.AAD{}).(string)
	session := ctx.Value(auth.Session{}).(string)

	req := v.(*ExecuteServiceRequest)

	if len(req.Address) == 0 {
		e := errors.New(errors.ErrInvalidAddress, nil)
		h.logger.Debug(ctx, "received empty address", log.MapFields{
			"call_type": "ExecuteServiceFailure",
			"session":   session,
		}, e)
		return nil, e
	}

	authReq := h.parseExecuteMessage(req)
	if err := h.verifier.Verify(ctx, authReq); err != nil {
		e := errors.New(errors.ErrFailedAADVerification, err)
		h.logger.Debug(ctx, "failed to verify AAD", log.MapFields{
			"call_type": "ExecuteServiceFailure",
			"session":   session,
			"err":       e,
		})
		return nil, e
	}

	// a context from an http request is cancelled after the response to the request is returned,
	// so a new context is needed to handle the asynchronous request
	id, err := h.client.ExecuteServiceAsync(context.Background(), backend.ExecuteServiceRequest{
		AAD:        aad,
		Address:    req.Address,
		Data:       req.Data,
		SessionKey: session,
	})
	if err != nil {
		h.logger.Debug(ctx, "failed to start request", log.MapFields{
			"call_type": "ExecuteServiceFailure",
			"address":   req.Address,
			"session":   session,
		}, err)
		return nil, err
	}

	return AsyncResponse{ID: id}, nil
}

func (h ServiceHandler) mapEvent(event backend.Event) Event {
	switch r := event.(type) {
	case backend.ErrorEvent:
		return ErrorEvent{
			ID:    r.ID,
			Cause: r.Cause,
		}
	case backend.ExecuteServiceResponse:
		return ExecuteServiceEvent{
			ID:      r.ID,
			Address: r.Address,
			Output:  r.Output,
		}
	case backend.DeployServiceResponse:
		return DeployServiceEvent{
			ID:      r.ID,
			Address: r.Address,
		}
	default:
		panic("received unexpected event type from polling service")
	}
}

// PollService polls the service response queue to retrieve available responses
func (h ServiceHandler) PollService(ctx context.Context, v interface{}) (interface{}, error) {
	session := ctx.Value(auth.Session{}).(string)
	req := v.(*PollServiceRequest)
	if req.Count == 0 {
		req.Count = 10
	}

	res, err := h.client.PollService(ctx, backend.PollServiceRequest{
		Offset:          req.Offset,
		Count:           req.Count,
		DiscardPrevious: req.DiscardPrevious,
		SessionKey:      session,
	})
	if err != nil {
		return nil, err
	}

	events := make([]Event, 0, len(res.Events))
	for _, r := range res.Events {
		events = append(events, h.mapEvent(r))
	}

	return PollServiceResponse{Offset: res.Offset, Events: events}, nil
}

// GetCode retrieves the source code associated with a service.
func (h ServiceHandler) GetCode(ctx context.Context, v interface{}) (interface{}, error) {
	req := v.(*GetCodeRequest)

	if len(req.Address) == 0 {
		err := errors.New(errors.ErrInvalidAddress, stderr.New("address field has not been set"))
		h.logger.Debug(ctx, "failed to start request", log.MapFields{
			"call_type": "GetCodeFailure",
			"address":   req.Address,
		}, err)
		return nil, err
	}

	res, err := h.client.GetCode(ctx, backend.GetCodeRequest{
		Address: req.Address,
	})

	if err != nil {
		h.logger.Debug(ctx, "request failed", log.MapFields{
			"call_type": "GetCodeFailure",
			"address":   req.Address,
		}, err)
		return nil, err
	}

	return GetCodeResponse{
		Address: res.Address,
		Code:    res.Code,
	}, nil
}

// GetPublicKey retrieves the public key associated with a service
// to allow the client to encrypt the data that serves as argument for
// a service deployment or service execution.
func (h ServiceHandler) GetPublicKey(ctx context.Context, v interface{}) (interface{}, error) {
	req := v.(*GetPublicKeyRequest)

	if len(req.Address) == 0 {
		err := errors.New(errors.ErrInvalidAddress, stderr.New("address field has not been set"))
		h.logger.Debug(ctx, "failed to start request", log.MapFields{
			"call_type": "GetPublicKeyFailure",
			"address":   req.Address,
		}, err)
		return nil, err
	}

	res, err := h.client.GetPublicKey(ctx, backend.GetPublicKeyRequest{
		Address: req.Address,
	})

	if err != nil {
		h.logger.Debug(ctx, "request failed", log.MapFields{
			"call_type": "GetPublicKeyFailure",
			"address":   req.Address,
		}, err)
		return nil, err
	}

	return GetPublicKeyResponse{
		Timestamp: res.Timestamp,
		Address:   res.Address,
		PublicKey: res.PublicKey,
		Signature: res.Signature,
	}, nil
}

func NewServiceHandler(services Services) ServiceHandler {
	if services.Client == nil {
		panic("Request must be provided as a service")
	}
	if services.Logger == nil {
		panic("Logger must be provided as a service")
	}

	return ServiceHandler{
		logger:   services.Logger.ForClass("service", "handler"),
		client:   services.Client,
		verifier: services.Verifier,
	}
}

// BindHandler binds the service handler to the provided
// HandlerBinder
func BindHandler(services Services, binder rpc.HandlerBinder) {
	handler := NewServiceHandler(services)

	binder.Bind("POST", "/v0/api/service/deploy", rpc.HandlerFunc(handler.DeployService),
		rpc.EntityFactoryFunc(func() interface{} { return &DeployServiceRequest{} }))
	binder.Bind("POST", "/v0/api/service/execute", rpc.HandlerFunc(handler.ExecuteService),
		rpc.EntityFactoryFunc(func() interface{} { return &ExecuteServiceRequest{} }))
	binder.Bind("POST", "/v0/api/service/poll", rpc.HandlerFunc(handler.PollService),
		rpc.EntityFactoryFunc(func() interface{} { return &PollServiceRequest{} }))
	binder.Bind("GET", "/v0/api/service/getCode", rpc.HandlerFunc(handler.GetCode),
		rpc.EntityFactoryFunc(func() interface{} { return &GetCodeRequest{} }))
	binder.Bind("GET", "/v0/api/service/getPublicKey", rpc.HandlerFunc(handler.GetPublicKey),
		rpc.EntityFactoryFunc(func() interface{} { return &GetPublicKeyRequest{} }))
	binder.Bind("POST", "/v0/api/service/getCode", rpc.HandlerFunc(handler.GetCode),
		rpc.EntityFactoryFunc(func() interface{} { return &GetCodeRequest{} }))
	binder.Bind("POST", "/v0/api/service/getPublicKey", rpc.HandlerFunc(handler.GetPublicKey),
		rpc.EntityFactoryFunc(func() interface{} { return &GetPublicKeyRequest{} }))
}
