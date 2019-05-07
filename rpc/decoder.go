package rpc

import (
	"encoding/json"
	"io"
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
func (e JsonDecoder) DecodeWithLimit(reader io.Reader, v interface{}, limit uint) error {
	limitReader := NewLimitReader(reader, limit)
	return e.Decode(&limitReader, v)
}

// NewLimitReader returns a new LimitReader
func NewLimitReader(reader io.Reader, limit uint) LimitReader {
	return LimitReader{
		limit:  limit,
		count:  0,
		reader: reader,
	}
}

// LimitReader is an io.Reader wrapper that ensures that
// no more than limit bytes are read from the reader
type LimitReader struct {
	limit  uint
	count  uint
	reader io.Reader
}

// Read is the implementation of Reader for LimitReader
func (r *LimitReader) Read(p []byte) (int, error) {
	if r.count >= r.limit {
		return 0, io.EOF
	}

	available := r.limit
	if uint(len(p)) < available {
		available = uint(len(p))
	}

	n, err := r.reader.Read(p[:available])
	r.count += uint(n)
	return n, err
}
