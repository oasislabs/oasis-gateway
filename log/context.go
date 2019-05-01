package log

import (
	"context"
)

type ContextKey string

const (
	ContextKeyTraceID ContextKey = "logContextKeyTraceID"
)

func GetTraceID(ctx context.Context) int64 {
	contextTraceID := ctx.Value(ContextKeyTraceID)
	if contextTraceID == nil {
		return -1
	}

	traceID, ok := contextTraceID.(int64)
	if !ok {
		return -1
	}

	return traceID
}
