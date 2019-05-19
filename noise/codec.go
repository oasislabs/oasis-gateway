package noise

import (
	"bytes"
	"io"

	"github.com/ugorji/go/codec"
)

// IncomingFrame is the frame used on top of the noise protocol to send messages
// and be able to multiplex them using the session ID as the identifier
type IncomingFrame struct {
	Payload codec.Raw `codec:"payload"`
}

// OutgoingFrame is the frame used on top of the noise protocol to send messages
// and be able to multiplex them using the session ID as the identifier
type OutgoingFrame struct {
	SessionID []byte `codec:"session"`
	Payload   []byte `codec:"payload"`
}

// MarshalFrame serializes a frame into bytes so that it can
// be sent to the remote endpoint
func MarshalFrame(f *OutgoingFrame) ([]byte, error) {
	buf := bytes.NewBuffer(make([]byte, len(f.Payload)+len(f.SessionID)+64))
	if err := SerializeFrame(buf, f); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// UnmarshalFrame deserializes a frame from bytes
func UnmarshalFrame(p []byte, f *IncomingFrame) error {
	buf := bytes.NewBuffer(p)
	return DeserializeFrame(buf, f)
}

// SerializeFrame serializes a frame into a writer
func SerializeFrame(w io.Writer, f *OutgoingFrame) error {
	return codec.NewEncoder(w, &codec.CborHandle{}).Encode(f)
}

// UnmarshalFrame deserializes a frame from a reader
func DeserializeFrame(r io.Reader, f *IncomingFrame) error {
	return codec.NewDecoder(r, &codec.CborHandle{}).Decode(f)
}
