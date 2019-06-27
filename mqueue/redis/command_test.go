package redis

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNextRequest(t *testing.T) {
	req := nextRequest{Key: "key"}

	assert.Equal(t, []string{"key"}, req.Keys())
	assert.Equal(t, []interface{}(nil), req.Args())
}

func TestInsertRequest(t *testing.T) {
	req := insertRequest{
		Offset:  1,
		Key:     "key",
		Content: "content",
		Type:    "type",
	}

	assert.Equal(t, []string{"key"}, req.Keys())
	assert.Equal(t, []interface{}{
		uint64(1),
		"type",
		"content",
	}, req.Args())
}

func TestRetrieveRequest(t *testing.T) {
	req := retrieveRequest{
		Offset: 1,
		Key:    "key",
		Count:  1,
	}

	assert.Equal(t, []string{"key"}, req.Keys())
	assert.Equal(t, []interface{}{
		uint64(1),
		uint(1),
	}, req.Args())
}

func TestDiscardRequest(t *testing.T) {
	req := discardRequest{
		KeepPrevious: true,
		Count:        1,
		Offset:       1,
		Key:          "key",
	}

	assert.Equal(t, []string{"key"}, req.Keys())
	assert.Equal(t, []interface{}{
		uint64(1),
		uint(1),
		true,
	}, req.Args())
}

func TestRemoveRequest(t *testing.T) {
	req := removeRequest{
		Key: "key",
	}

	assert.Equal(t, []string{"key"}, req.Keys())
	assert.Equal(t, []interface{}(nil), req.Args())
}
