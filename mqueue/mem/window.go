package mem

import (
	"github.com/oasislabs/developer-gateway/errors"
	"github.com/oasislabs/developer-gateway/mqueue/core"
	stderr "github.com/pkg/errors"
)

type element struct {
	Reserved bool
	Set      bool
	Offset   uint64
	Value    string
}

var (
	ErrFull              = stderr.New("window is full and cannot increase its size")
	ErrOffsetOutOfWindow = stderr.New("offset is out of the window's range")
	ErrOffsetNotReserved = stderr.New("offset is not reserved")
	ErrOffsetAlreadySet  = stderr.New("offset is already set")
)

// SlidingWindow is a sliding window of elements that keep track of
// only a contiguous subset (a window) of all the elements that
// supposed to exist in the structure.
//
// It enforces that the window can slide only when the elements
// that are going to be discarded have already been set.
type SlidingWindow struct {
	// maxSize is the maximum size the window can grow to. Once this
	// size is reached the only way to add elements to the window is
	// discarding old elements.
	maxSize uint

	// nextUnreservedIndex keep track of which is the next index
	// inside the window that is not reserved
	nextUnreservedIndex uint

	// nextUnsetIndex keeps track of which is the next index inside
	// the window that is not set
	nextUnsetIndex uint

	// offset is the base offset that is represented by the first element
	// in the window
	offset uint64

	// elements is the backing array for the window implementation
	elements []element
}

// SlidingWindowProps defines the behaviour of an SlidingWindow instance
type SlidingWindowProps struct {
	// InitialSize defines the initial size of the window
	InitialSize uint

	// MaxSize defines the maximum size the window can grow to
	MaxSize uint
}

// NewSlidingWindow creates a new instance of a SlidingWindow with the
// defined behaviour
func NewSlidingWindow(props SlidingWindowProps) SlidingWindow {
	if props.InitialSize == 0 {
		props.InitialSize = 16
	}

	if props.MaxSize == 0 {
		props.MaxSize = 1024
	}

	if props.InitialSize > props.MaxSize {
		props.MaxSize = props.InitialSize
	}

	return SlidingWindow{
		maxSize:             props.MaxSize,
		nextUnreservedIndex: 0,
		nextUnsetIndex:      0,
		offset:              0,
		elements:            make([]element, props.InitialSize),
	}
}

// Get returns all the elements in the range from offset to offset + count
func (w *SlidingWindow) Get(offset uint64, count uint) (core.Elements, errors.Err) {
	if offset < w.offset {
		offset = w.offset
	}

	res := core.Elements{Offset: w.Offset(), Elements: make([]core.Element, 0, 16)}
	index := uint(offset - w.offset)

	for i := index; i < count+index && i < uint(len(w.elements)); i++ {
		element := &w.elements[i]
		if element.Reserved && element.Set {
			res.Elements = append(res.Elements, core.Element{
				Offset: element.Offset,
				Value:  element.Value,
			})
		}
	}

	return res, nil
}

// ReserveNext reserves the next offset available in the
// window, or an error if it is not possible to provide
// a next offset because either the window cannot grow more
// or it cannot slide and discard elements that have not yet
// been set
func (w *SlidingWindow) ReserveNext() (uint64, errors.Err) {
	// initialization should ensure that len(w.elements) > 0
	if w.nextUnreservedIndex >= uint(len(w.elements)-1) {
		if n := w.makeRoom(); n == 0 {
			return 0, errors.New(errors.ErrQueueLimitReached, ErrFull)
		}
	}

	offset := uint64(w.nextUnreservedIndex) + w.offset

	if offset-w.offset >= uint64(len(w.elements)) {
		panic("offset is out of the window's range")
	}

	index := uint(offset - w.offset)
	if w.elements[index].Reserved || w.elements[index].Set {
		panic("attempt to reserve element in use")
	}

	w.elements[index].Reserved = true
	w.elements[index].Offset = offset
	w.nextUnreservedIndex = index + 1

	return offset, nil
}

// Offset returns the current base offset where the first element
// contained by the window is
func (w *SlidingWindow) Offset() uint64 {
	return w.offset
}

// Set sets the value for the element at offset `offset`. If the
// offset is not in the window's range or the element's state is not
// reserved or already set an error will be returned
func (w *SlidingWindow) Set(offset uint64, value string) errors.Err {
	if w.offset > offset || offset > w.offset+uint64(len(w.elements)) {
		return errors.New(errors.ErrOutOfRange, ErrOffsetOutOfWindow)
	}

	index := uint(offset - w.offset)
	if !w.elements[index].Reserved {
		return errors.New(errors.ErrInvalidStateChangeError, ErrOffsetNotReserved)
	}

	if w.elements[index].Set {
		return errors.New(errors.ErrInvalidStateChangeError, ErrOffsetAlreadySet)
	}

	w.elements[index].Set = true
	w.elements[index].Value = value

	if w.nextUnsetIndex == index {
		// update w.nextUnsetIndex to the next unset element
		for i := w.nextUnsetIndex; i < uint(len(w.elements)); i++ {
			if !w.elements[i].Set {
				w.nextUnsetIndex = i
				break
			}
		}
	}

	return nil
}

// Slide slides the window up to offset effectively discarding all
// the elements with a lower offset, and making room available for
// new offsets to be reserved and new elements to be set.
func (w *SlidingWindow) Slide(offset uint64) (uint, errors.Err) {
	return w.slide(offset)
}

// makeRoom either grows the window or slides it in order to
// make room for new elements. It returns the number of elements
// that have been made available
func (w *SlidingWindow) makeRoom() uint {
	currSize := uint(len(w.elements))
	if currSize < w.maxSize {
		// if it is possible grow the underlying slice
		// up to at most maxSize elements
		nextSize := currSize << 1
		if nextSize > w.maxSize {
			nextSize = w.maxSize
		}

		elements := make([]element, nextSize)
		copy(elements, w.elements)
		w.elements = elements
		return nextSize - currSize
	}

	return 0
}

func (w *SlidingWindow) slide(offset uint64) (uint, errors.Err) {
	if w.offset > offset {
		return 0, nil
	}

	if offset > w.offset+uint64(len(w.elements)) {
		return 0, errors.New(errors.ErrOutOfRange, ErrOffsetOutOfWindow)
	}

	limit := uint(offset - w.offset)
	if limit > w.nextUnsetIndex {
		limit = w.nextUnsetIndex
	}

	if limit == 0 {
		return 0, nil
	}

	copy(w.elements, w.elements[limit:])
	removed := uint(len(w.elements)) - limit
	for i := removed; i < uint(len(w.elements)); i++ {
		w.elements[i].Set = false
		w.elements[i].Reserved = false
		w.elements[i].Offset = 0
		w.elements[i].Value = ""
	}

	w.offset += uint64(limit)
	w.nextUnsetIndex -= limit
	w.nextUnreservedIndex -= limit

	return limit, nil
}
