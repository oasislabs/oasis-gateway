package core

import (
	"encoding/binary"
	"errors"
)

const (
	AD_LEN_OFFSET int = 15
	AD_OFFSET     int = 23
)

// Currently an unused verifier of the AAD in a transaction request.
// TODO: Complete implementation once the format of the request has been
// agreed upon.
func VerifyAAD(data string, expected string) error {
	if len(data) < 23 {
		return errors.New("Data is too short")
	}
	adLength := binary.BigEndian.Uint32([]byte(data[AD_LEN_OFFSET : AD_LEN_OFFSET+4]))
	if len(data) < AD_OFFSET+int(adLength) {
		return errors.New("Data is too short")
	}
	plaintext := data[AD_OFFSET : AD_OFFSET+int(adLength)]
	if plaintext != expected {
		return errors.New("AAD does not match expected")
	}
	return nil
}
