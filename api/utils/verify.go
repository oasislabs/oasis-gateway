package utils

import (
	"encoding/binary"
	"errors"
)

const (
	PT_LEN_OFFSET int = 18
	PT_OFFSET     int = 22
)

func VerifyAAD(data string, expected string) error {
	plaintextLength := binary.BigEndian.Uint32([]byte(data[PT_LEN_OFFSET : PT_LEN_OFFSET+4]))
	plaintext := data[PT_OFFSET : PT_OFFSET+int(plaintextLength)]
	if plaintext != expected {
		return errors.New("AAD does not match expected")
	}
	return nil
}
