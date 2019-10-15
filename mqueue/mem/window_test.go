package mem

import (
	"strconv"
	"testing"

	"github.com/oasislabs/oasis-gateway/mqueue/core"
	"github.com/stretchr/testify/assert"
)

func TestSlidingWindowInitialState(t *testing.T) {
	w := NewSlidingWindow(SlidingWindowProps{MaxSize: 16})
	assert.Equal(t, uint64(0), w.Offset())
}

func TestSlidingWindowSet(t *testing.T) {
	w := NewSlidingWindow(SlidingWindowProps{MaxSize: 16})

	next, err := w.ReserveNext()
	assert.Nil(t, err)
	assert.Equal(t, uint64(0), next)

	err = w.Set(next, "", "value")
	assert.Nil(t, err)

	els, err := w.Get(0, 1)
	assert.Nil(t, err)

	assert.Equal(t, core.Elements{Offset: 0, Elements: []core.Element{
		{Offset: uint64(0), Value: "value"},
	}}, els)
}

func TestSlidingWindowAlreadySet(t *testing.T) {
	w := NewSlidingWindow(SlidingWindowProps{MaxSize: 16})

	next, err := w.ReserveNext()
	assert.Nil(t, err)
	assert.Equal(t, uint64(0), next)

	err = w.Set(next, "", "value")
	assert.Nil(t, err)

	err = w.Set(next, "", "value")
	assert.Equal(t, ErrOffsetAlreadySet, err.Cause())
}

func TestSlidingWindowSetMultipleSlideFixed(t *testing.T) {
	w := NewSlidingWindow(SlidingWindowProps{MaxSize: 16})

	for i := 0; i < 1024; i++ {
		if i > 0 && i%10 == 0 {
			n, err := w.Slide(uint64(i))
			assert.Nil(t, err)
			assert.Equal(t, uint(10), n)
		}

		next, err := w.ReserveNext()
		assert.Nil(t, err)
		assert.Equal(t, uint64(i), next)

		err = w.Set(next, "", strconv.Itoa(i))
		assert.Nil(t, err)
	}

	els, err := w.Get(1009, 16)
	assert.Nil(t, err)
	assert.Equal(t, uint64(1020), els.Offset)
	assert.Equal(t, 4, len(els.Elements))

	for i := 0; i < 4; i++ {
		assert.Equal(t, strconv.Itoa(i+1020), els.Elements[i].Value)
	}
}

func TestSlidingWindowSetMultipleSlideWithGrowth(t *testing.T) {
	w := NewSlidingWindow(SlidingWindowProps{MaxSize: 512, InitialSize: 32})

	for i := 0; i < 1024; i++ {
		if i > 0 && i%32 == 0 {
			n, err := w.Slide(uint64(i))
			assert.Nil(t, err)
			assert.Equal(t, uint(32), n)
		}

		next, err := w.ReserveNext()
		assert.Nil(t, err)
		assert.Equal(t, uint64(i), next)

		err = w.Set(next, "", strconv.Itoa(i))
		assert.Nil(t, err)
	}

	els, err := w.Get(1009, 16)
	assert.Nil(t, err)
	assert.Equal(t, uint64(992), els.Offset)
	assert.Equal(t, 15, len(els.Elements))

	for i := 0; i < 15; i++ {
		assert.Equal(t, strconv.Itoa(i+1009), els.Elements[i].Value)
	}
}

func TestSlidingWindowSetNotReserved(t *testing.T) {
	w := NewSlidingWindow(SlidingWindowProps{
		MaxSize: 16,
	})

	err := w.Set(0, "", "value")
	assert.Equal(t, ErrOffsetNotReserved, err.Cause())
}

func TestSlidingWindowReserveToLimit(t *testing.T) {
	w := NewSlidingWindow(SlidingWindowProps{
		MaxSize: 16,
	})

	for i := 0; i < 15; i++ {
		next, err := w.ReserveNext()
		assert.Nil(t, err)
		assert.Equal(t, uint64(i), next)

		err = w.Set(next, "", strconv.Itoa(i))
		assert.Nil(t, err)
	}

	next, err := w.ReserveNext()
	assert.Equal(t, ErrFull, err.Cause())
	assert.Equal(t, uint64(0), next)
}

func TestSlidingWindowSlideAll(t *testing.T) {
	w := NewSlidingWindow(SlidingWindowProps{
		MaxSize: 16,
	})

	for i := 0; i < 15; i++ {
		next, err := w.ReserveNext()
		assert.Nil(t, err)
		assert.Equal(t, uint64(i), next)

		err = w.Set(next, "", strconv.Itoa(i))
		assert.Nil(t, err)
	}

	n, err := w.Slide(16)
	assert.Nil(t, err)
	assert.Equal(t, uint(15), n)
	assert.Equal(t, uint64(15), w.Offset())
}

func TestDiscardFirstElement(t *testing.T) {
	w := NewSlidingWindow(SlidingWindowProps{
		MaxSize: 16,
	})

	next, err := w.ReserveNext()
	assert.Nil(t, err)
	assert.Equal(t, uint64(0), next)

	n, err := w.Discard(0, 1)
	assert.Equal(t, uint(1), n)
	assert.Nil(t, err)
	assert.Equal(t, uint64(1), w.Offset())
}

func TestDiscardSingleElement(t *testing.T) {
	w := NewSlidingWindow(SlidingWindowProps{
		MaxSize: 16,
	})

	for i := 0; i < 2; i++ {
		next, err := w.ReserveNext()
		assert.Nil(t, err)
		assert.Equal(t, uint64(i), next)
	}

	n, err := w.Discard(1, 1)
	assert.Equal(t, uint(1), n)
	assert.Nil(t, err)
	assert.Equal(t, uint64(0), w.Offset())

	n, err = w.Discard(0, 1)
	assert.Equal(t, uint(1), n)
	assert.Nil(t, err)
	assert.Equal(t, uint64(2), w.Offset())
}

func TestDiscardMultipleElements(t *testing.T) {
	w := NewSlidingWindow(SlidingWindowProps{
		MaxSize: 16,
	})

	for i := 0; i < 10; i++ {
		next, err := w.ReserveNext()
		assert.Nil(t, err)
		assert.Equal(t, uint64(i), next)
	}

	n, err := w.Discard(0, 10)
	assert.Nil(t, err)
	assert.Equal(t, uint(10), n)
	assert.Equal(t, uint64(10), w.Offset())
}

func TestDiscardAndSlide(t *testing.T) {
	w := NewSlidingWindow(SlidingWindowProps{
		MaxSize: 16,
	})

	for i := 0; i < 10; i++ {
		next, err := w.ReserveNext()
		assert.Nil(t, err)
		assert.Equal(t, uint64(i), next)

		err = w.Set(next, "", strconv.Itoa(i))
		assert.Nil(t, err)
	}

	n, err := w.Discard(5, 5)
	assert.Nil(t, err)
	assert.Equal(t, uint(5), n)

	n, err = w.Slide(5)
	assert.Nil(t, err)
	assert.Equal(t, uint(10), n)
	assert.Equal(t, uint64(10), w.Offset())
}

func TestGetDiscarded(t *testing.T) {
	w := NewSlidingWindow(SlidingWindowProps{
		MaxSize: 16,
	})

	for i := 0; i < 10; i++ {
		next, err := w.ReserveNext()
		assert.Nil(t, err)
		assert.Equal(t, uint64(i), next)

		err = w.Set(next, "", strconv.Itoa(i))
		assert.Nil(t, err)
	}

	n, err := w.Discard(2, 5)
	assert.Nil(t, err)
	assert.Equal(t, uint(5), n)

	els, err := w.Get(0, 10)
	assert.Nil(t, err)
	assert.Equal(t, uint64(0), els.Offset)
	assert.Equal(t, []core.Element{
		{Offset: 0x0, Value: "0", Type: ""},
		{Offset: 0x1, Value: "1", Type: ""},
		{Offset: 0x7, Value: "7", Type: ""},
		{Offset: 0x8, Value: "8", Type: ""},
		{Offset: 0x9, Value: "9", Type: ""}}, els.Elements)
}
