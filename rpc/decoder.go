package rpc

import (
	"encoding/json"
	"io"

	stderr "github.com/pkg/errors"

	"github.com/oasislabs/oasis-gateway/rw"
)

// SimpleJsonDeserializer deserializes a json payload
// into an object
type SimpleJsonDeserializer struct {
	O interface{}
}

// Deserialize is the implementation of Deserialize for SimpleJsonDeserializer
func (s *SimpleJsonDeserializer) Deserialize(r io.Reader) error {
	return JsonDecoder{}.Decode(r, s.O)
}

// DeserializerFunc implementation of Deserializer for functions
type DeserializeFunc func(r io.Reader) error

// Deserialize is the implementation of Deserialize for DeserializeFunc
func (f DeserializeFunc) Deserialize(r io.Reader) error {
	return f(r)
}

// Deserializer decodes payloads and keeps the state of
// the deserialized object
type Deserializer interface {
	// Deserialize the contents of the reader into an object owned
	// by the deserializer
	Deserialize(r io.Reader) error
}

// Decoder for payloads
type Decoder interface {
	// Decode decodes the provided payload with its format from the
	// provided reader. In case of failure it is possible a partial
	// read has occurred
	Decode(r io.Reader, v interface{}) error
}

// JsonEncoder is a payload encoder that serializes to JSON
type JsonDecoder struct{}

// Decode is the implementation of Decoder for JsonDecoder
func (e JsonDecoder) Decode(reader io.Reader, v interface{}) error {
	return stderr.Wrap(json.NewDecoder(reader).Decode(v), "failed to decode json")
}

// DecodeWithLimit decodes the payload in the reader making sure not
// to exceed the limit provided
func (e JsonDecoder) DecodeWithLimit(reader io.Reader, v interface{}, props rw.ReadLimitProps) error {
	// the JSON decoder needs to receive an io.EOF from the reader to make sure
	// it finished reading from source
	props.ErrOnEOF = true
	limitReader := rw.NewLimitReader(reader, props)
	return e.Decode(&limitReader, v)
}
