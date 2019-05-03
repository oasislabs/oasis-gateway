package mem

import (
	"errors"

	"github.com/oasislabs/developer-gateway/mqueue/core"
)

type Element struct {
	Element *core.Element

	// Taken flag to specify if the element's offset is already
	// taken by a Next() operation
	Taken bool
}

// OrderedQueue is a simple queue that keeps the elements insterted
// in an ordered fashion. The OrderedQueue is implemented as a
// sliding window on a full sequence of events. The window grows up
// to a maximum number of elements, after which it slides, so the
// elements with the lowest offsets are lost to allow space for the
// elements with the new offsets
type OrderedQueue struct {
	// maxElements is the maximum number of elements that the list is
	// allowed to have. Is maxElements is to be exceeded by the next insertion,
	// the elements with the lowest offset will be removed to make up space
	maxElements uint

	// offset is the offset represented by the first element in the queue
	offset uint64

	// nextAvailableIndex keeps track of what the next available offset to
	// be taken is
	nextAvailableIndex uint

	// list keeps the memory space where the elements are stored
	elements []Element
}

// NewOrderedQueue creates a new instance of an OrderedQueue
func NewOrderedQueue(maxElements uint) *OrderedQueue {
	initial := uint(16)
	if maxElements < initial {
		initial = maxElements
	}

	return &OrderedQueue{
		maxElements:        maxElements,
		offset:             0,
		nextAvailableIndex: 0,
		elements:           make([]Element, initial),
	}
}

// Insert inserts an element to the OrderedQueue. In case there is not
// enough space remaining in the queue, the elements with the lowest
// offset are removed
func (q *OrderedQueue) Insert(element core.Element) error {
	if element.Offset < q.offset {
		return errors.New("attempt to insert element past the sequence offset being tracked")
	}

	neededLen := element.Offset - q.offset + 1
	if neededLen > uint64(len(q.elements)) {
		q.slideOffset(neededLen)
	}

	index := element.Offset - q.offset
	if q.elements[index].Taken && q.elements[index].Element != nil {
		return errors.New("attempt to insert element to an already set element")
	}

	q.elements[index] = Element{
		Element: &element,
		Taken:   true,
	}

	return nil
}

// Retrieve returns the list of elements that start at the offset and the list
// has at most count elements. All the elements between `offset` and `offset + count`
// that have not been initialized are filtered out
func (q *OrderedQueue) Retrieve(offset uint64, count uint) []*core.Element {
	if offset < q.offset {
		offset = q.offset
	}

	fromIndex := offset - q.offset
	if fromIndex > uint64(len(q.elements)) {
		return nil
	}

	result := make([]*core.Element, 0, 16)
	for i := int(fromIndex); uint(i) < count && i < len(q.elements); i++ {
		if q.elements[i].Element != nil {
			result = append(result, q.elements[i].Element)
		}
	}

	return result
}

// Discard discards all the elements that have an offset
// smaller or equal to the provided offset
func (q *OrderedQueue) Discard(offset uint64) {
	q.discard(offset)
}

func (q *OrderedQueue) Next() uint64 {
	curr := q.nextAvailableIndex

	// take the element at `curr` for the current request
	q.elements[curr].Taken = true

	// update q.nextAvailableIndex for the next request
	q.updateNextAvailableIndex(q.nextAvailableIndex)

	return q.offset + uint64(curr)
}

func (q *OrderedQueue) discard(offset uint64) {
	if q.offset > offset {
		return
	}

	copyFromIndex := offset - q.offset + 1
	if copyFromIndex < uint64(len(q.elements)) {
		copy(q.elements, q.elements[copyFromIndex:])

		for i := len(q.elements) - int(copyFromIndex) + 1; i < len(q.elements); i++ {
			q.elements[i].Taken = false
			q.elements[i].Element = nil
		}

		if uint64(q.nextAvailableIndex) < copyFromIndex {
			q.updateNextAvailableIndex(0)

		} else {
			q.updateNextAvailableIndex(q.nextAvailableIndex - uint(copyFromIndex))
		}

	} else {
		for i := range q.elements {
			q.elements[i].Taken = false
			q.elements[i].Element = nil
		}
		q.updateNextAvailableIndex(0)
	}

	q.offset = offset + 1
}

func (q *OrderedQueue) updateNextAvailableIndex(from uint) {
	// update q.nextAvailableIndex for the next request
	for i := from; i < uint(len(q.elements)); i++ {
		if !q.elements[i].Taken {
			q.nextAvailableIndex = i
			return
		}
	}

	// attempt to grow the underlying array of elements if
	// possible
	oldLen := uint(len(q.elements))
	newLen := uint(len(q.elements)) << 1
	if newLen >= q.maxElements {
		newLen = q.maxElements
	}

	if newLen > uint(len(q.elements)) {
		q.slideOffset(uint64(newLen))
		q.nextAvailableIndex = oldLen

	} else {
		q.discard(q.offset)
		q.nextAvailableIndex = uint(len(q.elements))
	}

	// an assert check to verify that the nextAvailableIndex has been
	// set correctly
	if q.elements[q.nextAvailableIndex].Taken {
		panic("set next available index to incorrect index")
	}
}

func (q *OrderedQueue) slideOffset(neededLen uint64) {
	copyFromIndex := uint(0)

	newLen := neededLen << 1
	if newLen > uint64(q.maxElements) {
		newLen = uint64(q.maxElements)
	}

	// newLen < q.maxElements = uint
	if newLen < neededLen {
		copyFromIndex = uint(neededLen - newLen)
	}

	q.grow(copyFromIndex, uint(newLen))
}

func (q *OrderedQueue) grow(copyFromIndex, size uint) {
	if size > q.maxElements {
		panic("attempt to grow ordered queue to more than maxElements size")
	}

	elements := q.elements
	needInitialize := true
	nextAvailableIndexFrom := uint(0)

	if size > uint(len(q.elements)) {
		needInitialize = false
		elements = make([]Element, size)
	}

	if copyFromIndex < uint(len(q.elements)) {
		needInitialize = true
		copy(elements, q.elements[copyFromIndex:])

		if q.nextAvailableIndex > copyFromIndex {
			nextAvailableIndexFrom = q.nextAvailableIndex - copyFromIndex
		}
	}

	if needInitialize {
		start := len(elements) - int(copyFromIndex)
		if start < 0 {
			start = 0
		}

		for i := start; i < len(elements); i++ {
			elements[i].Taken = false
			elements[i].Element = nil
		}
	}

	q.updateNextAvailableIndex(nextAvailableIndexFrom)
	q.elements = elements
	q.offset += uint64(copyFromIndex)
}
