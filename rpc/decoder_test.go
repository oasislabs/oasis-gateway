package rpc

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestJsonDecoderDecode(t *testing.T) {
	buffer := bytes.NewBufferString("{\"hamburger\":\"rare\",\"potato\":\"fried\"}\n")
	m := make(map[string]string)

	err := JsonDecoder{}.Decode(buffer, &m)

	assert.Nil(t, err)
	assert.Equal(t, map[string]string{
		"potato":    "fried",
		"hamburger": "rare",
	}, m)
}

func TestJsonDecoderDecodeWithLimit(t *testing.T) {
	buffer := bytes.NewBufferString("{\"hamburger\":\"rare\",\"potato\":\"fried\"}\n")
	m := make(map[string]string)

	err := JsonDecoder{}.DecodeWithLimit(buffer, &m, 1024)

	assert.Nil(t, err)
	assert.Equal(t, map[string]string{
		"potato":    "fried",
		"hamburger": "rare",
	}, m)
}

func TestJsonDecoderDecodeWithLimitTooMuchData(t *testing.T) {
	buffer := bytes.NewBufferString("{\"hamburger\":\"rare\",\"potato\":\"fried\"}\n")
	m := make(map[string]string)

	err := JsonDecoder{}.DecodeWithLimit(buffer, &m, 10)

	assert.Equal(t, "unexpected EOF", err.Error())
}
