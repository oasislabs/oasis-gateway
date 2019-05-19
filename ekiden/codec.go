package ekiden

import (
	"bytes"
	"errors"
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

type RequestMessage struct {
	Request RequestPayload `codec:"Request"`
}

// ResponsePayload is the representation of an ekiden
// response used for serialization/deserialization
type Body struct {
	// Success is the field that is set in case of a successful
	// response
	Success interface{} `codec:"Success"`

	// Error is the field that is set in case of a failed
	// response with information on the error's cause
	Error string `codec:"Error"`
}

type Response struct {
	Body Body `codec:"body"`
}

type ResponseMessage struct {
	Response Response `codec:"Response"`
}

func MarshalRequestMessage(req *RequestMessage) ([]byte, error) {
	buf := bytes.NewBuffer(make([]byte, 128))
	if err := SerializeRequestMessage(buf, req); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func SerializeRequestMessage(w io.Writer, req *RequestMessage) error {
	return codec.NewEncoder(w, &codec.CborHandle{}).Encode(req)
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
	buf := bytes.NewBuffer(p)
	return DeserializeResponse(buf, res)
}

// DeserializeResponse deserializes an ekiden response
func DeserializeResponse(r io.Reader, res *ResponsePayload) error {
	return codec.NewDecoder(r, &codec.CborHandle{}).Decode(res)
}
