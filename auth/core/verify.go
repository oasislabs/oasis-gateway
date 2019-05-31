package core

import (
	"encoding/binary"
	"errors"
)

const (
	cipherLengthOffset = 16
	aadLengthOffset    = 24
	cipherOffset       = 32
	nonceLength        = 5
)

// Verify the provided AAD in the transaction data with the expected AAD
// Transaction data is expected to be in the following format:
//   pk || cipher length || aad length || cipher || aad || nonce
//   - pk is expected to be 16 bytes
//   - cipher length and aad length are uint64 encoded in big endian
//   - nonce is expected to be 5 bytes
func Verify(data string, expectedAAD string) error {
	if len(data) < cipherOffset {
		return errors.New("Data is too short")
	}
	cipherLength := binary.BigEndian.Uint64([]byte(data[cipherLengthOffset:aadLengthOffset]))
	aadLength := binary.BigEndian.Uint64([]byte(data[aadLengthOffset:cipherOffset]))

	if len(data) < int(cipherOffset+cipherLength+aadLength+nonceLength) {
		return errors.New("Missing data")
	}

	aadOffset := cipherOffset + cipherLength
	aad := data[aadOffset : aadOffset+aadLength]

	if aad != expectedAAD {
		return errors.New("AAD does not match")
	}
	return nil
}
