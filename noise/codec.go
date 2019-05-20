package noise

import (
	"bytes"
	"errors"
	"io"

	"github.com/oasislabs/developer-gateway/rw"
	"github.com/ugorji/go/codec"
)

// RequestPayload is the representation of
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

// IncomingRequestMessage is the message type used when sending
// requests to the remote endpoint
type IncomingRequestMessage struct {
	Request codec.Raw `codec:"Request"`
}

// OutgoingRequestMessage is the message type used when sending
// requests to the remote endpoint
type OutgoingRequestMessage struct {
	Request RequestPayload `codec:"Request"`
}

// IncomingResponseMessage is the message type used when sending
// a response to a RequestMessage from the remote endpoint
type IncomingResponseMessage struct {
	Response codec.Raw `codec:"Response"`
}

// ResponseMessage defines the structure of a ResponseMessage.
// This struct should not be serialized directly, the helper
// (Serialize/Marshal)ResponseMessage should be used instead
type ResponseMessage struct {
	Response ResponsePayload
}

// OutgoingResponseMessage is the message type used when sending
// a response to a RequestMessage from the remote endpoint
type OutgoingResponseMessage struct {
	Response []byte `codec:"Response"`
}

// CloseMessage is the message type used when notifying
// the other end that the session will close
type CloseMessage struct{}

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

// ResponseBody is used to deserialize a received response
type ResponseBody struct {
	Body ResponsePayload `codec:"Body"`
}

// MarshalRequestMessage serializes an OutgoingRequestMessage
// to be sent to the remote endpoint
func MarshalRequestMessage(m *OutgoingRequestMessage) ([]byte, error) {
	buf := bytes.NewBuffer(make([]byte, 128))
	if err := SerializeRequestMessage(buf, m); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// SerializeRequestMessage serializes an OutgoingRequestMessage into a writer
func SerializeRequestMessage(w io.Writer, m *OutgoingRequestMessage) error {
	return codec.NewEncoder(w, &codec.CborHandle{}).Encode(m)
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

// SerializeIntoFrame serializes a reader into a frame
func SerializeIntoFrame(w io.Writer, r io.Reader, sessionID []byte) error {
	if buf, ok := r.(*bytes.Buffer); ok {
		return SerializeBufferIntoFrame(w, buf, sessionID)
	}

	buf := bytes.NewBuffer(make([]byte, 128))
	if _, err := rw.CopyWithLimit(buf, r, rw.ReadLimitProps{
		FailOnExceed: true,
		Limit:        65535,
	}); err != nil {
		return err
	}

	return SerializeBufferIntoFrame(w, buf, sessionID)
}

// SerializeBufferIntoFrame serializes the contents of a buffer into
// a RequestMessage
func SerializeBufferIntoFrame(w io.Writer, buf *bytes.Buffer, sessionID []byte) error {
	err := SerializeFrame(w, &OutgoingFrame{SessionID: sessionID, Payload: buf.Bytes()})
	// Consume all the buffer bytes since we read them regardless of whether
	// there's an error
	buf.Reset()
	return err
}

// UnmarshalFrame deserializes a frame from a reader
func DeserializeFrame(r io.Reader, f *IncomingFrame) error {
	return codec.NewDecoder(r, &codec.CborHandle{}).Decode(f)
}

// UnmarshalCloseMessage unmarshals a response message from the enclave.
func UnmarshalCloseMessage(p []byte, res *CloseMessage) error {
	var response []codec.Raw
	if err := codec.NewDecoderBytes(p, &codec.CborHandle{}).Decode(&response); err != nil {
		return err
	}

	if len(response) != 1 {
		return errors.New("close message should have one fields")
	}

	var t string
	if err := codec.NewDecoderBytes(response[0], &codec.CborHandle{}).Decode(&t); err != nil {
		return err
	}

	if t != "Close" {
		return errors.New("first field should be Close")
	}

	return nil
}

// DeserializeResponseMessage unmarshals a response message from the enclave.
func DeserializeResponseMessage(r io.Reader, res *ResponseMessage) error {
	handle := codec.CborHandle{}
	var response []codec.Raw
	if err := codec.NewDecoder(r, &handle).Decode(&response); err != nil {
		return err
	}

	if len(response) != 2 {
		return errors.New("response message should have two fields")
	}

	var t string
	if err := codec.NewDecoderBytes(response[0], &handle).Decode(&t); err != nil {
		return err
	}

	if t != "Response" {
		return errors.New("first field should be Response")
	}

	var body ResponseBody
	if err := codec.NewDecoderBytes(response[1], &codec.CborHandle{}).Decode(&body); err != nil {
		return err
	}

	res.Response = body.Body
	return nil
}
