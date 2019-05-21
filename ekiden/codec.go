package ekiden

import (
	"bytes"
	"io"

	"github.com/ugorji/go/codec"
)

// Address of a contract
type Address [32]byte

// RequestPayload is the representation of an ekiden
// request used for serialization/deserialization
type RequestPayload struct {
	// Method is the method that the request will invoke
	Method string `codec:"method"`

	// Args are the arguments for invocation
	Args interface{} `codec:"args"`
}

// ResponsePayload is the representation of an ekiden
// response used for serialization/deserialization
type ResponsePayload struct {
	// Success is the field that is set in case of a successful
	// response
	Success interface{} `codec:"Success"`

	// Error is the field that is set in case of a failed
	// response with information on the error's cause
	Error string `codec:"Error"`
}

// MarshalRequest serializes an ekiden request to he specified format
func MarshalRequest(req *RequestPayload) ([]byte, error) {
	buf := bytes.NewBuffer(make([]byte, 128))
	if err := SerializeRequest(buf, req); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// SerializeRequest serializes an ekiden request to he specified format
func SerializeRequest(w io.Writer, req *RequestPayload) error {
	return codec.NewEncoder(w, &codec.CborHandle{}).Encode(req)
}

// UnmarshalResponse deserializes an ekiden response
func UnmarshalResponse(p []byte, res *ResponsePayload) error {
	buf := bytes.NewBuffer(p)
	return DeserializeResponse(buf, res)
}

// DeserializeResponse deserializes an ekiden response
func DeserializeResponse(r io.Reader, res *ResponsePayload) error {
	return codec.NewDecoder(r, &codec.CborHandle{}).Decode(res)
}
