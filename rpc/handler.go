package rpc

import "context"

// Handler is the handler for RPC requests.
type Handler interface {
	// Handle handlers an rpc request and returns a response or error if needed.
	// Implementations should ensure that if a context is cancelled the request
	// handling should be halt gracefully and return an appropriate error.
	Handle(ctx context.Context, body interface{}) (interface{}, error)
}

// HandlerFunc is the type definition for a function to be able to act as a Handler
type HandlerFunc func(ctx context.Context, body interface{}) (interface{}, error)

// Handle is the implementation of the Handler interface for a HandlerFunc
func (h HandlerFunc) Handle(ctx context.Context, body interface{}) (interface{}, error) {
	return h(ctx, body)
}

// EntityFactory is an interface for types that build an object of some kind
type EntityFactory interface {
	Create() interface{}
}

// EntityFactoryFunc is a type to allow functions to act as a Factory
type EntityFactoryFunc func() interface{}

// Create is the implementation of Factory for FactoryFunc
func (f EntityFactoryFunc) Create() interface{} {
	return f()
}

// HandlerBinder binds a handler to a specific
// method and path
type HandlerBinder interface {
	// Bind binds a handler for a specific method and path, so that the handler
	// will be dispatched when method and path combination is provided
	Bind(method string, path string, handler Handler, factory EntityFactory)
}
