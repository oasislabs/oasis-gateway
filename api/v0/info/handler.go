package version

import (
	"context"

	"github.com/oasislabs/oasis-gateway/rpc"
)

// Deps are the dependencies expected by the VersionHandler
type Deps struct{}

// Handler is the handler to satisfy version related requests
type Handler struct{}

// NewHandler creates a new instance of a version handler
func NewHandler() Handler {
	return Handler{}
}

// GetVersion returns the version of the component
func (h Handler) GetVersion(ctx context.Context, v interface{}) (interface{}, error) {
	return &GetVersionResponse{
		Version: 0,
	}, nil
}

// GetVersion returns the version of the component
func (h Handler) GetSenders(ctx context.Context, v interface{}) (interface{}, error) {
	return &GetSendersResponse{
		Senders: make([]string, 0), // TODO
	}, nil
}

// BindHandler binds the version handler to the handler binder
func BindHandler(deps *Deps, binder rpc.HandlerBinder) {
	handler := NewHandler()

	binder.Bind("GET", "/v0/api/version", rpc.HandlerFunc(handler.GetVersion),
		rpc.EntityFactoryFunc(func() interface{} { return nil }))

	binder.Bind("GET", "/v0/api/getSenders", rpc.HandlerFunc(handler.GetSenders),
		rpc.EntityFactoryFunc(func() interface{} { return nil }))
}
