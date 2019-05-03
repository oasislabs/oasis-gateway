package log

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetTraceIDNotFound(t *testing.T) {
	traceID := GetTraceID(context.Background())
	assert.Equal(t, int64(-1), traceID)
}

func TestGetTraceIDNotInteger(t *testing.T) {
	ctx := context.WithValue(context.Background(), ContextKeyTraceID, "traceID")
	traceID := GetTraceID(ctx)
	assert.Equal(t, int64(-1), traceID)
}

func TestGetTraceIDInteger(t *testing.T) {
	ctx := context.WithValue(context.Background(), ContextKeyTraceID, int64(1234))
	traceID := GetTraceID(ctx)
	assert.Equal(t, int64(1234), traceID)
}
