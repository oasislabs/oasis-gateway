package exec

import (
	"context"
	"fmt"
	"math/big"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"

	"github.com/oasislabs/developer-gateway/conc"
	ethereum "github.com/oasislabs/developer-gateway/eth"
	"github.com/oasislabs/developer-gateway/log"
	"github.com/stretchr/testify/assert"
)

const address string = "0x6f6704e5a10332af6672e50b3d9754dc460dfa4d"

func initializeExecutor() (*TransactionExecutor, error) {
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

	executor := NewTransactionExecutor(
		privateKey,
		types.FrontierSigner{},
		0,
		pooledClient,
		logger.ForClass("wallet", "InternalWallet"),
	)

	return executor, nil
}

func TestTransactionNonce(t *testing.T) {
	executor, err := initializeExecutor()
	assert.Nil(t, err)

	var nonce uint64
	for i := 0; i < 10; i++ {
		nonce = executor.transactionNonce()
		assert.Equal(t, uint64(i), nonce)
	}
}

func TestExecutorSignTransaction(t *testing.T) {
	executor, err := initializeExecutor()
	assert.Nil(t, err)

	// Build a mock transaction
	gas := uint64(1000000)
	gasPrice := int64(1000000000)
	tx := types.NewTransaction(
		0,
		common.HexToAddress(address),
		big.NewInt(0),
		gas,
		big.NewInt(gasPrice),
		[]byte("data"),
	)

	tx, err = executor.signTransaction(tx)
	assert.Nil(t, err)

	V, R, S := tx.RawSignatureValues()
	assert.NotEqual(t, new(big.Int), V)
	assert.NotEqual(t, new(big.Int), R)
	assert.NotEqual(t, new(big.Int), S)
}
