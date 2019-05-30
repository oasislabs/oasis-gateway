package mem

import (
	"strconv"
	"testing"

	"github.com/oasislabs/developer-gateway/mqueue/core"
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

	err = w.Set(next, "value")
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

	err = w.Set(next, "value")
	assert.Nil(t, err)

	err = w.Set(next, "value")
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

		err = w.Set(next, strconv.Itoa(i))
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

		err = w.Set(next, strconv.Itoa(i))
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

	err := w.Set(0, "value")
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

		err = w.Set(next, strconv.Itoa(i))
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

		err = w.Set(next, strconv.Itoa(i))
		assert.Nil(t, err)
	}

	n, err := w.Slide(16)
	assert.Nil(t, err)
	assert.Equal(t, uint(15), n)
	assert.Equal(t, uint64(15), w.Offset())
}
