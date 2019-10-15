package event

import "github.com/oasislabs/oasis-gateway/rpc"

// AsyncResponse is the response returned by APIs that are asynchronous
// that return an ID that can be used by the user to receive and identify
// a response to the request when it is ready
type AsyncResponse struct {
	// ID to identify an asynchronous response. It uniquely identifies the
	// event and orders it in the sequence of events expected by the user
	ID uint64 `json:"id"`
}

// UnsubscribeRequest is used by the user to destroy a subscription to specific
// topics on the gateway.
type UnsubscribeRequest struct {
	// ID of the subscription to be destroyed
	ID uint64 `json:"id"`
}

// SubscribeRequest is used by the user to create a subscription to specific
// topics on the gateway.
type SubscribeRequest struct {
	// Events is the the list of event types the subscription intends
	// to be created for
	Events []string `json:"events"`

	// Filter is a url encoded list of query parameters that specify
	// filters to be applied to the subscribed topic
	Filter string `json:"filter"`
}

// SubscribeResponse returns an AsyncResponse which contains the ID
// that can be used to poll for notifications on the subscription
type SubscribeResponse AsyncResponse

// PollEventRequest is a request that allows the user to
// poll for events either from asynchronous requests or from
// subscriptions
type PollEventRequest struct {
	// ID is the id of the subscription returned in SubscribeResponse
	ID uint64 `json:"id"`

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

// PollEventResponse is the list of events that are returned for
// a subscription or a group of asynchronous requests
type PollEventResponse struct {
	// Offset is the current offset at which the provided list of events
	// start
	Offset uint64 `json:"offset"`

	// Events is the list of events that the server has starting from
	// the provided Offset
	Events []Event `json:"events"`
}

// Event is the interface that all events that can be returned from an
// EventPollingResponse need to return
type Event interface {
	// EventID to identify an asynchronous response. It uniquely identifies the
	// event and orders it in the sequence of events expected by the user
	EventID() uint64
}

// DataEvent is that event that can be polled by the user to poll
// for service logs for example, which they are a blob of data that the
// client knows how to manipulate
type DataEvent struct {
	// ID to identify the event itself within the sequence of events.
	ID uint64 `json:"id"`

	// Data is the blob of data related to this event
	Data string `json:"data"`

	// Topics is the list of topics to which the event refers
	Topics []string `json:"topics"`
}

// ErrorEvent is the event that can be polled by the user
// as a result to a a request that failed
type ErrorEvent struct {
	// ID to identify an asynchronous response. It uniquely identifies the
	// event and orders it in the sequence of events expected by the user
	ID uint64 `json:"id"`

	// Cause is the error that caused the event to failed
	Cause rpc.Error `json:"cause"`
}

// EventID is the implementation of Event for DataEvent
func (e DataEvent) EventID() uint64 {
	return e.ID
}

// EventID is the implementation of Event for ErrorEvent
func (e ErrorEvent) EventID() uint64 {
	return e.ID
}
