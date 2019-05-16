package ekiden

import cbor "bitbucket.org/bodhisnarkva/cbor/go"

// Address of a contract
type Address [32]byte

// RequestPayload is the representation of an ekiden
// request used for serialization/deserialization
type RequestPayload struct {
	// Method is the method that the request will invoke
	Method string `cbor:"method"`

	// Args are the arguments for invocation
	Args interface{} `cbor:"args"`
}

// ResponsePayload is the representation of an ekiden
// response used for serialization/deserialization
type ResponsePayload struct {
	// Success is the field that is set in case of a successful
	// response
	Success interface{} `cbor:"Success"`

	// Error is the field that is set in case of a failed
	// response with information on the error's cause
	Error string `cbor:"Error"`
}

// MarshalRequest serializes an ekiden request to he specified format
func MarshalRequest(req *RequestPayload) ([]byte, error) {
	return cbor.Dumps(req)
}

// UnmarshalResponse deserializes an ekiden response
func UnmarshalResponse(p []byte, res *ResponsePayload) error {
	return cbor.Loads(p, &res)
}
