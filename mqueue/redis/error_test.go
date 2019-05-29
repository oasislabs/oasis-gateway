package redis

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsErrSerialize(t *testing.T) {
	err := ErrSerialize{Cause: nil}
	assert.True(t, IsErrSerialize(err))
}

func TestIsErrRedisExec(t *testing.T) {
	err := ErrRedisExec{Cause: nil}
	assert.True(t, IsErrRedisExec(err))
}
