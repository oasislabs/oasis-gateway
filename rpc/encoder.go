package rpc

import (
	"encoding/json"
	"io"

	stderr "github.com/pkg/errors"
)

// SimpleJsonSerializer deserializes a json payload
// into an object
type SimpleJsonSerializer struct {
	O interface{}
}

// Serialize is the implementation of Serialize for SimpleJsonSerializer
func (s *SimpleJsonSerializer) Serialize(w io.Writer) error {
	return JsonEncoder{}.Encode(w, s.O)
}

// SerializerFunc implementation of Serializer for functions
type SerializeFunc func(w io.Writer) error

// Serialize is the implementation of Serialize for SerializeFunc
func (f SerializeFunc) Serialize(w io.Writer) error {
	return f(w)
}

// Serializer encodes payloads and keeps the state of
// the serialized object
type Serializer interface {
	// Serialize an object into the provided writer
	Serialize(w io.Writer) error
}

// Encoder for payloads
type Encoder interface {
	// Encode encodes the provided payload with its format to the
	// provided writer. In case of failure it is possible a partial
	// write of the serialization to the writer
	Encode(w io.Writer, v interface{}) error
}

// JsonEncoder is a payload encoder that serializes to JSON
type JsonEncoder struct{}

// Encode is the implementation of Encoder for JsonEncoder
func (e JsonEncoder) Encode(writer io.Writer, v interface{}) error {
	return stderr.Wrap(json.NewEncoder(writer).Encode(v), "failed to encode json")
}
