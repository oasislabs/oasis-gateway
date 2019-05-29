package core

import (
	"context"
	"fmt"
	"io/ioutil"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"

	"github.com/oasislabs/developer-gateway/conc"
	ethereum "github.com/oasislabs/developer-gateway/eth"
	"github.com/oasislabs/developer-gateway/log"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

var (
	ctx    = context.Background()
	logger = log.NewLogrus(log.LogrusLoggerProperties{
		Level:  logrus.DebugLevel,
		Output: ioutil.Discard,
	})
)

func initializeWallet() (Wallet, error) {
	privateKey, err := crypto.HexToECDSA(strings.Repeat("1", 64))
	if err != nil {
		return nil, fmt.Errorf("failed to initialize private key with error %s", err.Error())
	}

	ctx := context.Background()
	dialer := ethereum.NewUniDialer(ctx, "https://localhost:1111")
	pooledClient := ethereum.NewPooledClient(ethereum.PooledClientProps{
		Pool:        dialer,
		RetryConfig: conc.RandomConfig,
	})
	logger := log.NewLogrus(log.LogrusLoggerProperties{})

	wallet := &InternalWallet{
		privateKey: privateKey,
		signer:     types.FrontierSigner{},
		nonce:      0,
		client:     pooledClient,
		logger:     logger.ForClass("wallet", "InternalWallet"),
	}

	return wallet, nil
}

func TestAddress(t *testing.T) {
	wallet, err := initializeWallet()
	assert.Nil(t, err)

	address := wallet.Address().Hex()
	assert.Equal(t, "0x19E7E376E7C213B7E7e7e46cc70A5dD086DAff2A", address)
}

func TestTransactionClient(t *testing.T) {
  wallet, err := initializeWallet()
	assert.Nil(t, err)

	wallet.Address()
}

func TestTransactionNonce(t *testing.T) {
  wallet, err := initializeWallet()
	assert.Nil(t, err)

	var nonce uint64
	for i := 0; i < 10; i++ {
		nonce = wallet.TransactionNonce()
		assert.Equal(t, uint64(i), nonce)
	}
}
