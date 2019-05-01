package rpc

import "context"

// Handler is the handler for RPC requests.
type Handler interface {
	// Handle handlers an rpc request and returns a response or error if needed.
	// Implementations should ensure that if a context is cancelled the request
	// handling should be halt gracefully and return an appropriate error.
	Handle(ctx context.Context, body interface{}) (interface{}, error)
}

// HandleFunc is the type definition for a function to be able to act as a Handler
type HandleFunc func(ctx context.Context, body interface{}) (interface{}, error)

// Handle is the implementation of the Handler interface for a HandleFunc
func (h HandleFunc) Handle(ctx context.Context, body interface{}) (interface{}, error) {
	return h(ctx, body)
}

// Factory is an interface for types that build an object of some kind
type Factory interface {
	Create() interface{}
}

// FactoryFunc is a type to allow functions to act as a Factory
type FactoryFunc func() interface{}

// Create is the implementation of Factory for FactoryFunc
func (f FactoryFunc) Create() interface{} {
	return f()
}
