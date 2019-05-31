package core

import (
	"encoding/binary"
	"errors"
)

const (
	cipherLengthOffset = 16
	aadLengthOffset    = 24
	cipherOffset       = 32
)

// Verify the provided AAD in the transaction data with the expected AAD
func Verify(data string, expectedAAD string) error {
	if len(data) < cipherOffset {
		return errors.New("Data is too short")
	}
	cipherLength := binary.BigEndian.Uint64([]byte(data[cipherLengthOffset:aadLengthOffset]))
	aadLength := binary.BigEndian.Uint64([]byte(data[aadLengthOffset:cipherOffset]))

	if len(data) < int(cipherOffset+cipherLength+aadLength) {
		return errors.New("Failed to read AAD")
	}

	aadOffset := cipherOffset + cipherLength
	aad := data[aadOffset : aadOffset+aadLength]

	if aad != expectedAAD {
		return errors.New("AAD does not match")
	}
	return nil
}
