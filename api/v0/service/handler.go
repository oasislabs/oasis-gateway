package service

import (
	"context"

	auth "github.com/oasislabs/developer-gateway/auth/core"
	backend "github.com/oasislabs/developer-gateway/backend/core"
	"github.com/oasislabs/developer-gateway/log"
	"github.com/oasislabs/developer-gateway/rpc"
)

type Services struct {
	Logger  log.Logger
	Request *backend.RequestManager
}

// ServiceHandler implements the handlers for service management
type ServiceHandler struct {
	logger  log.Logger
	request *backend.RequestManager
}

// DeployService handles the deployment of new services
func (h ServiceHandler) DeployService(ctx context.Context, v interface{}) (interface{}, error) {
	_ = v.(*DeployServiceRequest)
	return nil, rpc.HttpNotImplemented(ctx, "not implemented")
}

// ExecuteService handle the execution of deployed services
func (h ServiceHandler) ExecuteService(ctx context.Context, v interface{}) (interface{}, error) {
	authID := ctx.Value(auth.ContextKeyAuthID).(string)

	req := v.(*ExecuteServiceRequest)

	// a context from an http request is cancelled after the response to the request is returned,
	// so a new context is needed to handle the asynchronous request
	id, err := h.request.ExecuteServiceAsync(context.Background(), backend.ExecuteServiceRequest{
		Address: req.Address,
		Data:    req.Data,
		Key:     authID,
	})
	if err != nil {
		h.logger.Debug(ctx, "failed to start request", log.MapFields{
			"call_type": "ExecuteServiceFailure",
			"err":       err.Error(),
		})
		return nil, rpc.HttpTooManyRequests(ctx, "too many requests to execute service received")
	}

	return AsyncResponse{ID: id}, nil
}

// PollService polls the service response queue to retrieve available responses
func (h ServiceHandler) PollService(ctx context.Context, v interface{}) (interface{}, error) {
	authID := ctx.Value(auth.ContextKeyAuthID).(string)

	req := v.(*PollServiceRequest)
	if req.Count == 0 {
		req.Count = 10
	}

	res, err := h.request.GetResponses(authID, req.Offset, req.Count)
	if err != nil {
		return nil, err
	}

	var events []Event
	for _, r := range res.Events {
		switch r := r.(type) {
		case backend.ErrorEvent:
			events = append(events, ErrorEvent{
				ID:    r.ID,
				Cause: r.Cause,
			})
		case backend.ExecuteServiceEvent:
			events = append(events, ExecuteServiceEvent{
				ID:      r.ID,
				Address: r.Address,
				Output:  r.Output,
			})
		default:
			panic("received unexpected event type from polling service")
		}
	}

	return PollServiceResponse{Offset: res.Offset, Events: events}, nil
}

// ListServices lists the service the client has access to
func (h ServiceHandler) ListServices(ctx context.Context, v interface{}) (interface{}, error) {
	_ = v.(*ListServiceRequest)
	return nil, rpc.HttpNotImplemented(ctx, "not implemented")
}

// GetPublicKeyService retrives the public key associated with a service
// to allow the client to encrypt the data that serves as argument for
// a service deployment or service execution.
func (h ServiceHandler) GetPublicKeyService(ctx context.Context, v interface{}) (interface{}, error) {
	_ = v.(*GetPublicKeyServiceRequest)
	return nil, rpc.HttpNotImplemented(ctx, "not implemented")
}

// BindHandler binds the service handler to the provided
// HandlerBinder
func BindHandler(services Services, binder rpc.HandlerBinder) {
	if services.Request == nil {
		panic("Request must be provided as a service")
	}
	if services.Logger == nil {
		panic("Logger must be provided as a service")
	}

	handler := ServiceHandler{
		logger:  services.Logger.ForClass("service", "handler"),
		request: services.Request,
	}

	binder.Bind("POST", "/v0/api/service/deploy", rpc.HandlerFunc(handler.DeployService),
		rpc.EntityFactoryFunc(func() interface{} { return &DeployServiceRequest{} }))
	binder.Bind("POST", "/v0/api/service/execute", rpc.HandlerFunc(handler.ExecuteService),
		rpc.EntityFactoryFunc(func() interface{} { return &ExecuteServiceRequest{} }))
	binder.Bind("POST", "/v0/api/service/poll", rpc.HandlerFunc(handler.PollService),
		rpc.EntityFactoryFunc(func() interface{} { return &PollServiceRequest{} }))
	binder.Bind("GET", "/v0/api/service/list", rpc.HandlerFunc(handler.ListServices),
		rpc.EntityFactoryFunc(func() interface{} { return &ListServiceRequest{} }))
	binder.Bind("GET", "/v0/api/service/getPublicKey", rpc.HandlerFunc(handler.GetPublicKeyService),
		rpc.EntityFactoryFunc(func() interface{} { return &GetPublicKeyServiceRequest{} }))
}
