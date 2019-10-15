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
	_ = v.(*GetVersionRequest)
	return &GetVersionResponse{
		Version: 0,
	}, nil
}

// BindHandler binds the version handler to the handler binder
func BindHandler(deps *Deps, binder rpc.HandlerBinder) {
	handler := NewHandler()

	binder.Bind("GET", "/v0/api/version", rpc.HandlerFunc(handler.GetVersion),
		rpc.EntityFactoryFunc(func() interface{} { return nil }))
}
