package stats

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCounterInit(t *testing.T) {
	c := Counter{}

	assert.Equal(t, uint64(0), c.Value())
}

func TestCounterIncr(t *testing.T) {
	c := Counter{}

	for i := 1; i <= 10; i++ {
		assert.Equal(t, uint64(i), c.Incr())
	}

	assert.Equal(t, uint64(10), c.Value())
}

func TestCounterGroupGetFound(t *testing.T) {
	group := NewCounterGroup("counter1", "counter2")

	counter := group.Get("counter1")
	assert.Equal(t, uint64(0), counter.Value())
}

func TestCounterGroupGetNotFound(t *testing.T) {
	group := NewCounterGroup("counter1", "counter2")

	counter := group.Get("notFound")
	assert.Equal(t, uint64(0), counter.Value())
}

func TestCounterGroupIncrNotFound(t *testing.T) {
	group := NewCounterGroup("counter1", "counter2")

	assert.Equal(t, uint64(1), group.Incr("notFound1"))
	assert.Equal(t, uint64(2), group.Incr("notFound2"))
}

func TestCounterGroupIncrFound(t *testing.T) {
	group := NewCounterGroup("counter1", "counter2")

	counter := group.Get("counter1")

	assert.Equal(t, uint64(1), group.Incr("counter1"))
	assert.Equal(t, uint64(2), group.Incr("counter1"))
	assert.Equal(t, uint64(2), counter.Value())
}

func TestCounterGroupStats(t *testing.T) {
	group := NewCounterGroup("counter1", "counter2")

	assert.Equal(t, uint64(1), group.Incr("counter1"))
	assert.Equal(t, uint64(2), group.Incr("counter1"))
	assert.Equal(t, uint64(1), group.Incr("counter2"))
	assert.Equal(t, uint64(1), group.Incr("notFound1"))
	assert.Equal(t, uint64(2), group.Incr("notFound2"))
	assert.Equal(t, uint64(3), group.Incr("notFound3"))

	assert.Equal(t, map[string]interface{}{
		"counter1":  uint64(2),
		"counter2":  uint64(1),
		"undefined": uint64(3),
	}, group.Stats())
}
