package ekiden

// SubmitRequest is the request to submit a transaction to
// ekiden
type SubmitRequest struct {
	// Method to be invoked for the request
	Method string

	// RuntimeID is the ID of the runtime that will handle the request
	RuntimeID []byte

	// Data is the RLP encoded representation of the data that
	// is sent
	Data []byte
}

// SubmitResponse is the runtime's response to a submit request
type SubmitResponse struct {
	// Result contains the resulting value of a successful response
	Result interface{}
}
