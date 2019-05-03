package rpc

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseContextString(t *testing.T) {
	traceID := ParseTraceID("traceID")
	assert.Equal(t, int64(-1), traceID)
}

func TestParseContextEmpty(t *testing.T) {
	traceID := ParseTraceID("")
	assert.Equal(t, int64(-1), traceID)
}

func TestParseContextInteger(t *testing.T) {
	traceID := ParseTraceID("12345")
	assert.Equal(t, int64(12345), traceID)
}
