package rpc

import (
	"context"
	"net/http"
	"strconv"
	"time"
)

var ContextStatsTimeTrackKey = "rpcContextStatsTimeTrackKey"

type TimeTrack struct {
	Start int64
	Stop  int64
}

// ParseTraceID parses a traceID from a string and in case of failure
// it returns a default -1
func ParseTraceID(s string) int64 {
	if len(s) == 0 {
		return -1
	}

	value, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return -1
	}

	return value
}

// WithTimeTrack adds the latency tracking to the http request's
// context
func WithTimeTrack(req *http.Request) *http.Request {
	latency := TimeTrack{Start: time.Now().UnixNano()}
	ctx := context.WithValue(req.Context(), ContextStatsTimeTrackKey, &latency)
	return req.WithContext(ctx)
}

// MustFinishTimeTrack returns TimeTrack from a context if the latency
// is present in the request's context. It panics if the context
// does not have Latency
func MustFinishTimeTrack(req *http.Request) *TimeTrack {
	t, ok := FinishTimeTrack(req)
	if !ok {
		panic("TimeTrack not present in request's context")
	}
	return t
}

// FinishTimeTrack returns latency from a context if TimeTrack
// is present in the request's context
func FinishTimeTrack(req *http.Request) (*TimeTrack, bool) {
	v := req.Context().Value(ContextStatsTimeTrackKey)
	if t, ok := v.(*TimeTrack); ok {
		t.Stop = time.Now().UnixNano()
		return t, true
	}

	return nil, false
}
