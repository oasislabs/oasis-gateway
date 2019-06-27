package event

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDataEventEventID(t *testing.T) {
	assert.Equal(t, uint64(1), DataEvent{ID: 1}.EventID())
}

func TestErrorEventEventID(t *testing.T) {
	assert.Equal(t, uint64(1), ErrorEvent{ID: 1}.EventID())
}
