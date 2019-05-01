package rpc

import (
	"encoding/json"
	"io"
)

// Encoder for payloads
type Encoder interface {
	// Encode encodes the provided payload with its format to the
	// provided writer. In case of failure it is possible a partial
	// write of the serialization to the writer
	Encode(writer io.Writer, v interface{}) error
}

// JsonEncoder is a payload encoder that serializes to JSON
type JsonEncoder struct{}

// Encode is the implementation of Encoder for JsonEncoder
func (e JsonEncoder) Encode(writer io.Writer, v interface{}) error {
	return json.NewEncoder(writer).Encode(v)
}
