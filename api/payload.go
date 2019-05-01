package api

// Permission defines the level of permission a user has
// on a specific service.
type Permission int

const (
	Read             Permission = 1
	Write            Permission = 2
	ReadWrite        Permission = 3
	Execute          Permission = 4
	ReadExecute      Permission = 5
	WriteExecute     Permission = 6
	ReadWriteExecute Permission = 7
)

// ServicePermission defines the service abstraction of a contract
type ServicePermission struct {
	// Address is s the unique address that identifies the service,
	// is generated when a service is deployed and it can be used
	// for service execution
	Address string `json:"address"`

	// Level of permission granted to the user on the service
	Permission int `json:"permission"`
}

// AsyncResponse is the response returned by APIs that are asynchronous
// that return an ID that can be used by the user to receive and identify
// a response to the request when it is ready
type AsyncResponse struct {
	// ID to identifiy an asynchronous response. It uniquely identifies the
	// event and orders it in the sequence of events expected by the user
	ID int64 `json:"id"`
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

// ExecuteServiceRequest is isused by the user to trigger a service
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

// SubscribeRequest is used by the user to create a subscription to specific
// topics on the gateway.
type SubscribeRequest struct {
	// Topic is the the topic for the subscription
	Topic string `json:"topic"`

	// Filter is a url encoded list of query parameters that specifiy
	// filters to be applied to the subscribed topic
	Filter string `json:"filter"`
}

// SubscribeResponse returns an AsyncResponse which contains the ID
// that can be used to poll for notifications on the subscription
type SubscribeResponse AsyncResponse

// GetPublicKeyServiceRequest is a request to retrieve the public key
// associated with a specific service
type GetPublicKeyServiceRequest struct {
	// Address is s the unique address that identifies the service,
	// is generated when a service is deployed and it can be used
	// for service execution
	Address string `json:"address"`
}

// GetPublicKeyServiceResponse is the response in which the public key
// associated with the contract is provided
type GetPublicKeyServiceResponse struct {
	// Address is s the unique address that identifies the service,
	// is generated when a service is deployed and it can be used
	// for service execution
	Address string `json:"address"`

	// PublicKey associated to the service
	PublicKey string `json:"publicKey"`
}

// EventPollingRequest is a request that allows the user to
// poll for events either from asynchronous requests or from
// subscriptions
type EventPollingRequest struct {
	// Offset at which events need to be provided. Events are all ordered
	// with sequence numbers and it is up to the client to specifiy which
	// events it wants to receive from an offset in the sequence
	Offset int `json:"offset"`

	// Count for the number of items the client would prefer to receive
	// at most from a single response
	Count int `json:"count"`

	// DiscardPrevious allows the client to define whether the server should
	// discard all the events that have a sequence number lower than the offer
	DiscardPrevious bool `json:"discardPrevious"`
}

// EventPollingResponse is the list of events that are returned for
// a subscription or a group of asynchronous requests
type EventPollingResponse struct {
	// Events is the list of events that the server has starting from
	// the provided Offset
	Events []Event `json:"events"`

	// Offset is the current offset at which the provided list of events
	// start
	Offset int `json:"offset"`
}

// Event is the interface that all events that can be returned from an
// EventPollingResponse need to return
type Event interface {
	// ID to identifiy an asynchronous response. It uniquely identifies the
	// event and orders it in the sequence of events expected by the user
	ID() int64
}

// DataEvent is that event that can be polled by the user to poll
// for service logs for example, which they are a blob of data that the
// client knows how to manipulate
type DataEvent struct {
	// ID to identify the event itself withint the sequence of events.
	ID int64 `json:"id"`

	// Data is the blob of data related to this event
	Data string `json:"data"`
}

// ServiceExecutionEvent is the event that can be polled by the user
// as a result to a ServiceExecutionRequest
type ServiceExecutionEvent struct {
	// ID to identify an asynchronous response. It uniquely identifies the
	// event and orders it in the sequence of events expected by the user
	ID int64 `json:"id"`

	// Address is s the unique address that identifies the service,
	// is generated when a service is deployed and it can be used
	// for service execution
	Address string `json:"address"`

	// Output generated by the service at the end of its execution
	Output string `json:"output"`
}

// ServiceDeployEvent is the event that can be polled by the user
// as a result to a ServiceExecutionRequest
type ServiceDeployEvent struct {
	// ID to identifiy an asynchronous response. It uniquely identifies the
	// event and orders it in the sequence of events expected by the user
	ID int64 `json:"id"`

	// Address is s the unique address that identifies the service,
	// is generated when a service is deployed and it can be used
	// for service execution
	Address string `json:"address"`
}
