package core

import (
	"context"

	"github.com/oasislabs/oasis-gateway/stats"
)

// Element represents an element of the OrderedQueue
type Element struct {
	// Offset is the offset of the element within the sequence
	// of elements that is stored in the queue
	Offset uint64

	// Value is the arbitrary value stored by the queue
	Value string

	// Type allows the user to set a string to identify the
	// type of the value stored. It may be useful when
	// deserializing
	Type string
}

// Elements is an ordered set of elements
type Elements struct {
	// Offset is the base offset from which the elements are taken. That is
	// if Offset is N, that means that the Elements array starts with
	// offset N, and if element N is not present in the array it means that it
	// is still pending
	Offset uint64

	// Elements is the collection of elements starting from offset Offset
	Elements []Element
}

// InsertRequest is the request to insert elements into a queue
type InsertRequest struct {
	// Key unique identifier of the queue
	Key string

	// Element to be inserted to the queue
	Element Element
}

// RetrieveRequest to request the queue to all the
// elements in the sequence starting at Offset
// and has at most Count elements
type RetrieveRequest struct {
	// Key unique identifier of the queue
	Key string

	// Offset at which the retrieving window should start. Elements
	// with at a lower offset will not be returned
	Offset uint64

	// Count is the number of elements at most that will be returned as
	// part of the request
	Count uint
}

// DiscardRequest to request the queue to discard all the
// elements in the queue up to Offset.
//
// The request creates a slice of elements to be discarded, which
// is,
// - if KeepPrevious == true, [offset, offset + count].
// - if KeepPrevious == false, [0, offset+count].
type DiscardRequest struct {
	// KeepPrevious when set to true will tell the mqueue to keep the
	// elements in the queue with a lower offset than Offset
	KeepPrevious bool

	// Count is the number of elements after the Offset that will also be
	// discarded
	Count uint

	// Offset that defines the offset for the request.
	Offset uint64

	// Key unique identifier of the queue
	Key string
}

// NextRequest to request the next offset available
// in the queue that can be inserted
type NextRequest struct {
	// Key unique identifier of the queue
	Key string
}

// RemoveRequest to ask to destroy the queue identified
// by the provided key
type RemoveRequest struct {
	// Key unique identifier of the queue
	Key string
}

// ExistsRequest to ask to destroy the queue identified
// by the provided key
type ExistsRequest struct {
	// Key unique identifier of the queue
	Key string
}

// MQueue is an interface to a messaging queue service that
// provides the basic operations for a simple publish
// subscribe mechanism in which the clients manage the offsets
// for each queue they have.
type MQueue interface {
	// Name is a human readable identifier
	Name() string

	// Stats returns collected health metrics for the queue
	Stats() stats.Metrics

	// Insert inserts the element to the provided offset.
	Insert(context.Context, InsertRequest) error

	// Retrieve all available elements from the
	// messaging queue after the provided offset
	Retrieve(context.Context, RetrieveRequest) (Elements, error)

	// Discard all elements that have a prior or equal
	// offset to the provided offset
	Discard(context.Context, DiscardRequest) error

	// Next element offset that can be used for the queue.
	Next(context.Context, NextRequest) (uint64, error)

	// Remove the queue and associated resources with the key
	Remove(context.Context, RemoveRequest) error

	// Exists returns true if the key exists
	Exists(context.Context, ExistsRequest) (bool, error)
}
