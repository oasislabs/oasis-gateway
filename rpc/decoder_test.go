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

	assert.Nil(t, err)
	assert.Equal(t, map[string]string{
		"potato":    "fried",
		"hamburger": "rare",
	}, m)
}
