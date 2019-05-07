package rpc

import (
	"bytes"
	"io/ioutil"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestJsonEncoderEncode(t *testing.T) {
	buffer := bytes.NewBufferString("")

	err := JsonEncoder{}.Encode(buffer, map[string]string{
		"potato":    "fried",
		"hamburger": "rare",
	})
	assert.Nil(t, err)

	p, err := ioutil.ReadAll(buffer)
	assert.Nil(t, err)
	assert.Equal(t, "{\"hamburger\":\"rare\",\"potato\":\"fried\"}\n", string(p))
}
