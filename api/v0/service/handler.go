package service

import (
	"context"

	auth "github.com/oasislabs/developer-gateway/auth/core"
	"github.com/oasislabs/developer-gateway/log"
	"github.com/oasislabs/developer-gateway/rpc"
)

type Services struct {
	Logger  log.Logger
	Request *rpc.RequestManager
}

// ServiceHandler implements the handlers for service management
type ServiceHandler struct {
	logger  log.Logger
	request *rpc.RequestManager
}

// DeployService handles the deployment of new services
func (h ServiceHandler) DeployService(ctx context.Context, v interface{}) (interface{}, error) {
	_ = v.(*DeployServiceRequest)
	return nil, rpc.HttpNotImplemented(ctx, "not implemented")
}

// ExecutService handle the execution of deployed services
func (h ServiceHandler) ExecuteService(ctx context.Context, v interface{}) (interface{}, error) {
	authID := ctx.Value(auth.ContextKeyAuthID).(string)

	req := v.(*ExecuteServiceRequest)
	id, err := h.request.StartRequest(authID, req)
	if err != nil {
		h.logger.Debug(ctx, "failed to start request", log.MapFields{
			"call_type": "ExecuteServiceFailure",
			"err":       err.Error(),
		})
		return nil, rpc.HttpTooManyRequests(ctx, "too many requests to execute service received")
	}

	return AsyncResponse{ID: id}, nil
}

// ListServices lists the service the client has access to
func (H ServiceHandler) ListServices(ctx context.Context, v interface{}) (interface{}, error) {
	_ = v.(*ListServiceRequest)
	return nil, rpc.HttpNotImplemented(ctx, "not implemented")
}

// GetPublicKeyService retrives the public key associated with a service
// to allow the client to encrypt the data that serves as argument for
// a service deployment or service execution.
func (H ServiceHandler) GetPublicKeyService(ctx context.Context, v interface{}) (interface{}, error) {
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
	binder.Bind("GET", "/v0/api/service/list", rpc.HandlerFunc(handler.ListServices),
		rpc.EntityFactoryFunc(func() interface{} { return &ListServiceRequest{} }))
	binder.Bind("GET", "/v0/api/service/getPublicKey", rpc.HandlerFunc(handler.GetPublicKeyService),
		rpc.EntityFactoryFunc(func() interface{} { return &GetPublicKeyServiceRequest{} }))
}
