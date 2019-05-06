package rpc

import "strconv"

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
