package oauth

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"testing"

	"github.com/oasislabs/developer-gateway/auth/core"
	auth "github.com/oasislabs/developer-gateway/auth/core"
	"github.com/stretchr/testify/assert"
)

const (
	expectedAAD = "expectedAAD"
	pk          = "0000000000000000"
	cipertext   = "00000000000000000000000000000000"
	nonce       = "00000"

	dataFormat = "%s%s%s%s%s%s"
)

func generateData(pk, cipher, aad, nonce string) (string, error) {
	cipherLengthBuf := new(bytes.Buffer)
	aadLengthBuf := new(bytes.Buffer)
	if err := binary.Write(cipherLengthBuf, binary.BigEndian, uint64(len(cipher))); err != nil {
		return "", err
	}
	if err := binary.Write(aadLengthBuf, binary.BigEndian, uint64(len(aad))); err != nil {
		return "", err
	}
	return fmt.Sprintf(
		dataFormat,
		pk,
		cipherLengthBuf.String(),
		aadLengthBuf.String(),
		cipher,
		aad,
		nonce), nil
}

func TestVerifyOK(t *testing.T) {
	data, err := generateData(pk, cipertext, expectedAAD, nonce)
	assert.Nil(t, err)
	ctx := context.WithValue(context.Background(), core.AAD{}, expectedAAD)

	err = GoogleOauth{}.Verify(ctx, auth.AuthRequest{Data: data, PK: []byte(pk), AAD: []byte(expectedAAD)})
	assert.Nil(t, err)
}

func TestVerifyMissingLengths(t *testing.T) {
	data, err := generateData(pk, cipertext, expectedAAD, nonce)
	assert.Nil(t, err)
	ctx := context.WithValue(context.Background(), core.AAD{}, expectedAAD)

	err = GoogleOauth{}.Verify(ctx, auth.AuthRequest{Data: data[0:28]})
	assert.Error(t, err)
	assert.Equal(t, "AAD does not match", err.Error())
}

func TestVerifyMissingNonce(t *testing.T) {
	data, err := generateData(pk, cipertext, expectedAAD, nonce)
	assert.Nil(t, err)
	ctx := context.WithValue(context.Background(), core.AAD{}, expectedAAD)

	err = GoogleOauth{}.Verify(ctx, auth.AuthRequest{Data: data[:len(data)-5]})
	assert.Error(t, err)
	assert.Equal(t, "AAD does not match", err.Error())
}

func TestVerifyMismatchedAAD(t *testing.T) {
	data, err := generateData(pk, cipertext, expectedAAD, nonce)
	assert.Nil(t, err)
	ctx := context.WithValue(context.Background(), core.AAD{}, "wrongAAD")

	err = GoogleOauth{}.Verify(ctx, auth.AuthRequest{Data: data, PK: []byte(pk), AAD: []byte(expectedAAD)})
	assert.Error(t, err)
	assert.Equal(t, "AAD does not match", err.Error())
}
