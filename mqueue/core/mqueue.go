package core

import (
	"context"

	"github.com/oasislabs/developer-gateway/errors"
)

// Element represents an element of the OrderedQueue
type Element struct {
	// Offset is the offset of the element within the sequence
	// of elements that is stored in the queue
	Offset uint64

	// Value is the arbitrary value stored by the queue
	Value interface{}
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
	Key     string
	Element Element
}

// RetrieveRequest to request the queue to all the
// elements in the sequence starting at Offset
// and has at most Count elements
type RetrieveRequest struct {
	// Key unique identifier of the queue
	Key    string
	Offset uint64
	Count  uint
}

// DiscardRequest to request the queue to discard all the
// elements in the queue up to Offset
type DiscardRequest struct {
	// Key unique identifier of the queue
	Key    string
	Offset uint64
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

// MQueue is an interface to a messaging queue service that
// provides the basic operations for a simple publish
// subscribe mechanism in which the clients manage the offsets
// for each queue they have.
type MQueue interface {
	// Insert inserts the element to the provided offset.
	Insert(context.Context, InsertRequest) errors.Err

	// Retrieve all available elements from the
	// messaging queue after the provided offset
	Retrieve(context.Context, RetrieveRequest) (Elements, errors.Err)

	// Discard all elements that have a prior or equal
	// offset to the provided offset
	Discard(context.Context, DiscardRequest) errors.Err

	// Next element offset that can be used for the queue.
	Next(context.Context, NextRequest) (uint64, errors.Err)

	// Remove the queue and associated resources with the key
	Remove(context.Context, RemoveRequest) errors.Err
}
