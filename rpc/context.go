package rpc

import "strconv"

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
