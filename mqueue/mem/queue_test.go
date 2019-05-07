package mem

import (
	"testing"

	"github.com/oasislabs/developer-gateway/mqueue/core"
	"github.com/stretchr/testify/assert"
)

func TestQueueInsertElement(t *testing.T) {
	queue := NewOrderedQueue(16)

	err := queue.Insert(core.Element{Value: 1, Offset: 0})
	assert.Nil(t, err)

	els := queue.Retrieve(0, 1)
	assert.Equal(t, 1, len(els.Elements))

	assert.Equal(t, uint64(0), els.Offset)
	assert.Equal(t, 1, els.Elements[0].Value.(int))
}

func TestQueueInsertAlreadyPresent(t *testing.T) {
	queue := NewOrderedQueue(16)

	err := queue.Insert(core.Element{Value: 1, Offset: 0})
	assert.Nil(t, err)

	err = queue.Insert(core.Element{Value: 1, Offset: 0})
	assert.Equal(t, "attempt to insert element to an already set element", err.Error())
}

func TestQueueInsertMultipleElement(t *testing.T) {
	queue := NewOrderedQueue(16)

	for i := 0; i < 1024; i++ {
		err := queue.Insert(core.Element{Value: i, Offset: uint64(i)})
		assert.Nil(t, err)
	}

	els := queue.Retrieve(0, 1024)

	assert.Equal(t, uint64(1008), els.Offset)
	assert.Equal(t, 16, len(els.Elements))
	for i := 0; i < 16; i++ {
		assert.Equal(t, i+1008, els.Elements[i].Value.(int))
	}
}

func TestQueueNextInsert(t *testing.T) {
	queue := NewOrderedQueue(16)

	next := queue.Next()
	assert.Equal(t, uint64(0), next)

	err := queue.Insert(core.Element{Value: 1, Offset: uint64(1024)})
	assert.Nil(t, err)

	next = queue.Next()
	assert.Equal(t, uint64(1009), next)
}

func TestQueueNextAllAvailable(t *testing.T) {
	queue := NewOrderedQueue(32)

	for i := 0; i < 16; i++ {
		assert.Equal(t, uint64(i), queue.Next())
	}
}

func TestQueueNextDiscardNotLast(t *testing.T) {
	queue := NewOrderedQueue(16)

	err := queue.Insert(core.Element{Value: 1, Offset: uint64(1024)})
	assert.Nil(t, err)

	queue.Discard(1023)

	next := queue.Next()
	assert.Equal(t, uint64(1025), next)
}

func TestQueueNextDiscardUpToLast(t *testing.T) {
	queue := NewOrderedQueue(16)

	err := queue.Insert(core.Element{Value: 1, Offset: uint64(1024)})
	assert.Nil(t, err)

	queue.Discard(1024)

	next := queue.Next()
	assert.Equal(t, uint64(1025), next)
}
