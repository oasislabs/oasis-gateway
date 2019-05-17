package ekiden

import (
	"errors"

	cbor "bitbucket.org/bodhisnarkva/cbor/go"
	"github.com/ugorji/go/codec"
)

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

type Frame struct {
	SessionID []byte `cbor:"session"`
	Payload   []byte `cbor:"payload"`
}

type RequestMessage struct {
	Request RequestPayload `cbor:"Request"`
}

// ResponsePayload is the representation of an ekiden
// response used for serialization/deserialization
type Body struct {
	// Success is the field that is set in case of a successful
	// response
	Success interface{} `cbor:"Success"`

	// Error is the field that is set in case of a failed
	// response with information on the error's cause
	Error string `cbor:"Error"`
}

type Response struct {
	Body Body `cbor:"body"`
}

type ResponseMessage struct {
	Response Response `cbor:"Response"`
}

// MarshalFrame serializes an ekiden frame to the specified format
func MarshalFrame(frame *Frame) ([]byte, error) {
	return cbor.Dumps(frame)
}

func MarshalRequestMessage(req *RequestMessage) ([]byte, error) {
	return cbor.Dumps(req)
}

// MarshalRequest serializes an ekiden request to he specified format
func MarshalRequest(req *RequestPayload) ([]byte, error) {
	return cbor.Dumps(req)
}

// UnmarshalResponseMessage unmarshals a response message from the enclave.
func UnmarshalResponseMessage(p []byte, res *ResponseMessage) error {
	var response []codec.Raw
	if err := codec.NewDecoderBytes(p, &codec.CborHandle{}).Decode(&response); err != nil {
		return err
	}

	if len(response) != 2 {
		return errors.New("response message should have two fields")
	}

	var t string
	if err := codec.NewDecoderBytes(response[0], &codec.CborHandle{}).Decode(&t); err != nil {
		return err
	}

	if t != "Response" {
		return errors.New("first field should be Response")
	}

	var body Body
	if err := codec.NewDecoderBytes(response[1], &codec.CborHandle{}).Decode(&body); err != nil {
		return err
	}

	res.Response = Response{Body: body}
	return nil
}

// UnmarshalResponse deserializes an ekiden response
func UnmarshalResponse(p []byte, res *ResponsePayload) error {
	return cbor.Loads(p, res)
}
