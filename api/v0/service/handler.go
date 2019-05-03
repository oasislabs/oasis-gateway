package service

import (
	"context"

	"github.com/oasislabs/developer-gateway/rpc"
)

// ServiceHandler implements the handlers for service management
type ServiceHandler struct{}

// DeployService handles the deployment of new services
func (h ServiceHandler) DeployService(ctx context.Context, v interface{}) (interface{}, error) {
	_ = v.(*DeployServiceRequest)
	return nil, rpc.HttpNotImplemented(ctx, "not implemented")
}

// ExecutService handle the execution of deployed services
func (h ServiceHandler) ExecuteService(ctx context.Context, v interface{}) (interface{}, error) {
	_ = v.(*ExecuteServiceRequest)
	return nil, rpc.HttpNotImplemented(ctx, "not implemented")
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
func BindHandler(binder rpc.HandlerBinder) {
	handler := ServiceHandler{}

	binder.Bind("POST", "/v0/api/service/deploy", rpc.HandlerFunc(handler.DeployService),
		rpc.EntityFactoryFunc(func() interface{} { return &DeployServiceRequest{} }))
	binder.Bind("POST", "/v0/api/service/execute", rpc.HandlerFunc(handler.ExecuteService),
		rpc.EntityFactoryFunc(func() interface{} { return &ExecuteServiceRequest{} }))
	binder.Bind("GET", "/v0/api/service/list", rpc.HandlerFunc(handler.ListServices),
		rpc.EntityFactoryFunc(func() interface{} { return &ListServiceRequest{} }))
	binder.Bind("GET", "/v0/api/service/getPublicKey", rpc.HandlerFunc(handler.GetPublicKeyService),
		rpc.EntityFactoryFunc(func() interface{} { return &GetPublicKeyServiceRequest{} }))
}
