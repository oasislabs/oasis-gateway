package rpc

// Error is the response returned by the server when it fails
// to satisfy a request
type Error struct {
	// TraceID is the identifier that allows to trace the request across the
	// call trace. It may be useful for debugging
	TraceID int64 `json:"traceId"`

	// ErrorCode is a unique identifier for the error that can be used to identify
	// the particular type of error encountered
	ErrorCode int `json:"errorCode"`

	// Description is a human readable description of the error that occurred
	// to aid the client in debugging
	Description string `json:"description"`
}

// Error is the implementation of go's error interface for Error
func (e Error) Error() string {
	return e.Description
}
