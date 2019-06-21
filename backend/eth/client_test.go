package eth

import (
	"context"
	"crypto/ecdsa"
	"errors"
	"fmt"
	"io/ioutil"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/oasislabs/developer-gateway/backend/core"
	backend "github.com/oasislabs/developer-gateway/backend/core"
	"github.com/oasislabs/developer-gateway/callback/callbacktest"
	"github.com/oasislabs/developer-gateway/eth"
	"github.com/oasislabs/developer-gateway/eth/ethtest"
	"github.com/oasislabs/developer-gateway/log"
	"github.com/oasislabs/developer-gateway/tx"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

var (
	Context           = context.TODO()
	PrivateKey string = "17be884d0713e46a983fe65900c0ee0f45696cee60e5611ebc80841cfad407b7"
	Logger            = log.NewLogrus(log.LogrusLoggerProperties{
		Level:  logrus.DebugLevel,
		Output: ioutil.Discard,
	})
)

func GetPrivateKey() *ecdsa.PrivateKey {
	privateKey, err := crypto.HexToECDSA(PrivateKey)
	if err != nil {
		panic(fmt.Sprintf("failed to create private key: %s", err.Error()))
	}

	return privateKey
}

func NewClientWithMock() (*Client, error) {
	mockclient := &ethtest.MockClient{}
	mockcallbacks := &callbacktest.MockClient{}

	mockclient.On("BalanceAt",
		mock.AnythingOfType("*context.emptyCtx"),
		mock.AnythingOfType("common.Address"),
		mock.AnythingOfType("*big.Int")).
		Return(big.NewInt(1), nil)

	executor, err := tx.NewExecutor(Context, &tx.ExecutorServices{
		Logger:    Logger,
		Client:    mockclient,
		Callbacks: mockcallbacks,
	}, &tx.ExecutorProps{PrivateKeys: []*ecdsa.PrivateKey{GetPrivateKey()}})
	if err != nil {
		return nil, err
	}

	return NewClientWithDeps(Context, &ClientDeps{
		Logger:   Logger,
		Client:   mockclient,
		Executor: executor,
	}), nil
}

func TestGetPublicKeyInvalidAddress(t *testing.T) {
	client, err := NewClientWithMock()
	assert.Nil(t, err)

	_, err = client.GetPublicKey(Context, backend.GetPublicKeyRequest{
		Address: "0x",
	})
	assert.Error(t, err)
	assert.Equal(t, "[2006] error code InputError with desc Provided invalid address.", err.Error())
}

func TestGetPublicKeyErr(t *testing.T) {
	client, err := NewClientWithMock()
	assert.Nil(t, err)

	client.client.(*ethtest.MockClient).On("GetPublicKey",
		mock.AnythingOfType("*context.emptyCtx"),
		mock.AnythingOfType("common.Address")).
		Return(eth.PublicKey{}, errors.New("error"))

	_, err = client.GetPublicKey(Context, backend.GetPublicKeyRequest{
		Address: "0x0000000000000000000000000000000000000000",
	})
	assert.Error(t, err)
	assert.Equal(t, "[1000] error code InternalError with desc Internal Error. Please check the status of the service. with cause failed to get public key error", err.Error())
}

func TestGetPublicKeyOK(t *testing.T) {
	client, err := NewClientWithMock()
	assert.Nil(t, err)

	client.client.(*ethtest.MockClient).On("GetPublicKey",
		mock.AnythingOfType("*context.emptyCtx"),
		mock.AnythingOfType("common.Address")).
		Return(eth.PublicKey{
			Timestamp: 1234,
			PublicKey: "0x6f6704e5a10332af6672e50b3d9754dc460dfa4d",
			Signature: "0x6f6704e5a10332af6672e50b3d9754dc460dfa4d",
		}, nil)

	pk, err := client.GetPublicKey(Context, backend.GetPublicKeyRequest{
		Address: "0x0000000000000000000000000000000000000000",
	})

	assert.Nil(t, err)
	assert.Equal(t, core.GetPublicKeyResponse{
		Timestamp: 1234,
		Address:   "0x0000000000000000000000000000000000000000",
		PublicKey: "0x6f6704e5a10332af6672e50b3d9754dc460dfa4d",
		Signature: "0x6f6704e5a10332af6672e50b3d9754dc460dfa4d",
	}, pk)
}
