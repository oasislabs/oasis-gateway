package stats

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIntWindowAddLessMax(t *testing.T) {
	w := NewIntWindow(16)

	for i := 0; i < 8; i++ {
		w.Add(int64(i))
	}

	assert.Equal(t, Metrics{
		"avg": float64(3.5),
	}, w.Stats())
}

func TestIntWindowAddMax(t *testing.T) {
	w := NewIntWindow(16)

	for i := 0; i < 16; i++ {
		w.Add(int64(i))
	}

	assert.Equal(t, Metrics{
		"avg": float64(7.5),
	}, w.Stats())
}

func TestIntWindowAddMoreMax(t *testing.T) {
	w := NewIntWindow(16)

	for i := 0; i < 24; i++ {
		w.Add(int64(i))
	}

	assert.Equal(t, Metrics{
		"avg": float64(15.5),
	}, w.Stats())
}

func TestIntWindowAddExceedCap(t *testing.T) {
	w := NewIntWindow(16)

	for i := 0; i < 73; i++ {
		w.Add(int64(i))
	}

	assert.Equal(t, Metrics{
		"avg": float64(64.5),
	}, w.Stats())
}
