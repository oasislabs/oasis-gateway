package core

// Element represents an element of the OrderedQueue
type Element struct {
	// Value is the arbitrary value stored by the queue
	Value interface{}

	// Offset is the offset of the element within the sequence
	// of elements that is stored in the queue
	Offset uint64
}

// MQueue is an interface to a messaging queue service that
// provides the basic operations for a simple publish
// subscribe mechanism in which the clients manage the offsets
// for each queue they have.
type MQueue interface {
	// Insert inserts the element to the provided offset.
	Insert(key string, element Element) error

	// Retrieve all available elements from the
	// messaging queue after the provided offset
	Retrieve(key string, offset uint64, count uint) ([]*Element, error)

	// Discard all elements that have a prior or equal
	// offset to the provided offset
	Discard(key string, offset uint64) error

	// Next element offset that can be used for the queue.
	Next(key string) (uint64, error)
}
