package info

import (
	"context"
	"io/ioutil"
	"testing"

	ethereum "github.com/ethereum/go-ethereum/common"
	"github.com/oasislabs/oasis-gateway/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

var Context = context.TODO()

var Logger = log.NewLogrus(log.LogrusLoggerProperties{
	Output: ioutil.Discard,
})

type MockClient struct {
	mock.Mock
}

func (c *MockClient) Senders() []ethereum.Address {
	return []ethereum.Address{
		ethereum.HexToAddress("0x01234567890abcdefa17a5dAfF8dC9b86eE04773"),
		ethereum.HexToAddress("0x0a51514857B379A521C580a10822Fd8A7aC491A0"),
	}
}

func createInfoHandler() InfoHandler {
	return NewInfoHandler(Services{
		Logger: Logger,
		Client: &MockClient{},
	})
}

func TestGetVersion(t *testing.T) {
	h := createInfoHandler()

	res, err := h.GetVersion(Context, nil)

	assert.Nil(t, err)
	assert.Equal(t, &GetVersionResponse{
		Version: 0,
	}, res)
}

func TestGetSenders(t *testing.T) {
	h := createInfoHandler()

	res, err := h.GetSenders(Context, nil)

	assert.Nil(t, err)
	assert.Equal(t, &GetSendersResponse{
		Addresses: []string{
			"0x01234567890abcdEfa17A5daFf8Dc9b86ee04773",
			"0x0A51514857b379a521C580a10822fD8a7aC491A0",
		},
	}, res)
}
