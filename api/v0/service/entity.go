package service

import "github.com/oasislabs/developer-gateway/rpc"

// ServicePermission defines the service abstraction of a contract
// TODO(stan): we still need to define how permissions will be managed
type ServicePermission struct {
	// Address is the unique address that identifies the service,
	// is generated when a service is deployed and it can be used
	// for service execution
	Address string `json:"address"`
}

// AsyncResponse is the response returned by APIs that are asynchronous
// that return an ID that can be used by the user to receive and identify
// a response to the request when it is ready
type AsyncResponse struct {
	// ID to identify an asynchronous response. It uniquely identifies the
	// event and orders it in the sequence of events expected by the user
	ID uint64 `json:"id"`
}

// ListServiceRequest for the list services API, whose purpose is
// to list the services that are available to a particular client based
// on the authorization policies defined
type ListServiceRequest struct {
	// Filter is a url encoded list of query parameters that specifiy
	// filters to be applied to the request logic
	Filter string `json:"filter"`
}

// ListServiceResponse for the list services API. Returns the list
// of services a user is authorized to see and the permissions
// the user has on each service
type ListServiceResponse struct {
	// Services is the list of permissions the user has on each service
	// it is authorized to read
	Services []ServicePermission `json:"services"`
}

// ExecuteServiceRequest is is used by the user to trigger a service
// execution. A client is always subscribed to a subcription with
// topic "service" from which the client can retrieve the asynchronous
// results to this request
type ExecuteServiceRequest struct {
	// Data is a blob of data that the user wants to pass to the service
	// as argument
	Data string `json:"data"`

	// Address where the service can be found
	Address string `json:"address"`
}

// ExecuteServiceResponse is an asynchronous response that will be obtained
// using the polling mechanims
type ExecuteServiceResponse AsyncResponse

// DeployServiceRequest is issued by the user to trigger a service
// execution. A client is always subscribed to a subcription with
// topic "service" from which the client can retrieve the asynchronous
// results to this request
type DeployServiceRequest struct {
	// Data is a blob of data that the user wants to pass as argument for
	// the deployment of a service
	Data string `json:"data"`
}

// DeployServiceResponse is an asynchronous response that will be obtained
// using the polling mechanism
type DeployServiceResponse AsyncResponse

// GetPublicKeyServiceRequest is a request to retrieve the public key
// associated with a specific service
type GetPublicKeyServiceRequest struct {
	// Address is the unique address that identifies the service,
	// is generated when a service is deployed and it can be used
	// for service execution
	Address string `json:"address"`
}

// GetPublicKeyServiceResponse is the response in which the public key
// associated with the contract is provided
type GetPublicKeyServiceResponse struct {
	// Address is the unique address that identifies the service,
	// is generated when a service is deployed and it can be used
	// for service execution
	Address string `json:"address"`

	// PublicKey associated to the service
	PublicKey string `json:"publicKey"`
}

// PollServiceRequest is a request that allows the user to
// poll for events either from asynchronous responses
type PollServiceRequest struct {
	// Offset at which events need to be provided. Events are all ordered
	// with sequence numbers and it is up to the client to specify which
	// events it wants to receive from an offset in the sequence
	Offset uint64 `json:"offset"`

	// Count for the number of items the client would prefer to receive
	// at most from a single response
	Count uint `json:"count"`

	// DiscardPrevious allows the client to define whether the server should
	// discard all the events that have a sequence number lower than the offer
	DiscardPrevious bool `json:"discardPrevious"`
}

// Event is an interface for types that can be fetched by polling on
// a service
type Event interface {
	// EventID is the ID that uniquely identifies the event and it is found
	// inside a sequence of events
	EventID() uint64
}

// PollServiceResponse returns a list of asynchronous responses
// the client requested
type PollServiceResponse struct {
	// Offset is the base offset the requests were got from
	Offset uint64 `json:"offset"`

	// Events is the list of events that the server has starting from
	// the provided Offset
	Events []Event `json:"events"`
}

// ExecuteServiceEvent is the event that can be polled by the user
// as a result to a ServiceExecutionRequest
type ExecuteServiceEvent struct {
	// ID to identify an asynchronous response. It uniquely identifies the
	// event and orders it in the sequence of events expected by the user
	ID uint64 `json:"id"`

	// Address is the unique address that identifies the service,
	// is generated when a service is deployed and it can be used
	// for service execution
	Address string `json:"address"`

	// Output generated by the service at the end of its execution
	Output string `json:"output"`
}

// DeployServiceEvent is the event that can be polled by the user
// as a result to a ServiceExecutionRequest
type DeployServiceEvent struct {
	// ID to identifiy an asynchronous response. It uniquely identifies the
	// event and orders it in the sequence of events expected by the user
	ID uint64 `json:"id"`

	// Address is the unique address that identifies the service,
	// is generated when a service is deployed and it can be used
	// for service execution
	Address string `json:"address"`
}

// ErrorEvent is the event that can be polled by the user
// as a result to a request that failed
type ErrorEvent struct {
	// ID to identifiy an asynchronous response. It uniquely identifies the
	// event and orders it in the sequence of events expected by the user
	ID uint64 `json:"id"`

	// Cause is the error that caused the event to failed
	Cause rpc.Error
}

// EventID is the implementation of rpc.Event for ExecuteServiceEvent
func (e ExecuteServiceEvent) EventID() uint64 {
	return e.ID
}

// EventID is the implementation of rpc.Event for DeployServiceEvent
func (e DeployServiceEvent) EventID() uint64 {
	return e.ID
}

// EventID is the implementation of rpc.Event for ErrorEvent
func (e ErrorEvent) EventID() uint64 {
	return e.ID
}
