package exec

import (
	"context"
	stderr "errors"
	"fmt"
	"math/big"
	"strings"
	"testing"

	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"

	"github.com/oasislabs/developer-gateway/eth"
	"github.com/oasislabs/developer-gateway/log"
	"github.com/stretchr/testify/assert"
)

const address string = "0x6f6704e5a10332af6672e50b3d9754dc460dfa4d"

type ExecutorMockClient struct {
	SendTransactionCount int
}

func NewExecutorMockClient() *ExecutorMockClient {
	return &ExecutorMockClient{
		SendTransactionCount: 0,
	}
}

func (m *ExecutorMockClient) EstimateGas(ctx context.Context, msg ethereum.CallMsg) (uint64, error) {
	return 0, nil
}

func (m *ExecutorMockClient) GetPublicKey(ctx context.Context, address common.Address) (eth.PublicKey, error) {
	return eth.PublicKey{}, nil
}

func (m *ExecutorMockClient) NonceAt(ctx context.Context, address common.Address) (uint64, error) {
	return 1, nil
}

func (m *ExecutorMockClient) SendTransaction(ctx context.Context, tx *types.Transaction) (eth.SendTransactionResponse, error) {
	m.SendTransactionCount++
	if tx.Nonce() != 1 {
		return eth.SendTransactionResponse{}, stderr.New("Invalid transaction nonce")
	}
	return eth.SendTransactionResponse{
		Status: StatusOK,
		Output: "Success",
		Hash:   "Some hash",
	}, nil
}

func (m *ExecutorMockClient) SubscribeFilterLogs(ctx context.Context, query ethereum.FilterQuery, c chan<- types.Log) (ethereum.Subscription, error) {
	return nil, nil
}

func (m *ExecutorMockClient) TransactionReceipt(ctx context.Context, txHash common.Hash) (*types.Receipt, error) {
	receipt := types.Receipt{
		ContractAddress: common.HexToAddress(strings.Repeat("0", 20)),
	}
	return &receipt, nil
}

func initializeExecutor() (*TransactionExecutor, error) {
	privateKey, err := crypto.HexToECDSA(strings.Repeat("1", 64))
	if err != nil {
		return nil, fmt.Errorf("failed to initialize private key with error %s", err.Error())
	}

	logger := log.NewLogrus(log.LogrusLoggerProperties{})

	executor := NewTransactionExecutor(
		privateKey,
		types.FrontierSigner{},
		0,
		NewExecutorMockClient(),
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

func TestExecuteTransactionNoAddressBadNonce(t *testing.T) {
	executor, err := initializeExecutor()
	assert.Nil(t, err)

	req := executeRequest{
		ID:      0,
		Address: "",
		Data:    []byte(""),
	}
	_, err = executor.executeTransaction(context.TODO(), req)
	assert.Nil(t, err)
	assert.Equal(t, 2, executor.client.(*ExecutorMockClient).SendTransactionCount)
}

func TestExecuteTransactionAddressBadNonce(t *testing.T) {
	executor, err := initializeExecutor()
	assert.Nil(t, err)

	req := executeRequest{
		ID:      0,
		Address: strings.Repeat("0", 20),
		Data:    []byte(""),
	}
	_, err = executor.executeTransaction(context.TODO(), req)
	assert.Nil(t, err)
	assert.Equal(t, 2, executor.client.(*ExecutorMockClient).SendTransactionCount)
}
