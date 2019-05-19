package rpc

import (
	"encoding/json"
	"io"

	"github.com/oasislabs/developer-gateway/rw"
)

// Decoder for payloads
type Decoder interface {
	// Decode decodes the provided payload with its format from the
	// provided reader. In case of failure it is possible a partial
	// read has occurred
	Decode(reader io.Reader, v interface{}) error
}

// JsonEncoder is a payload encoder that serializes to JSON
type JsonDecoder struct{}

// Decode is the implementation of Decoder for JsonDecoder
func (e JsonDecoder) Decode(reader io.Reader, v interface{}) error {
	return json.NewDecoder(reader).Decode(v)
}

// DecodeWithLimit decodes the payload in the reader making sure not
// to exceed the limit provided
func (e JsonDecoder) DecodeWithLimit(reader io.Reader, v interface{}, props rw.ReadLimitProps) error {
	limitReader := rw.NewLimitReader(reader, props)
	return e.Decode(&limitReader, v)
}
