package exec

import (
	"context"
	stderr "errors"
	"fmt"
	"math/big"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"

	"github.com/oasislabs/developer-gateway/eth"
	"github.com/oasislabs/developer-gateway/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

const address string = "0x6f6704e5a10332af6672e50b3d9754dc460dfa4d"

func implementMockClient(client *MockClient) {
	client.On("EstimateGas", mock.AnythingOfType("*context.emptyCtx"), mock.AnythingOfType("ethereum.CallMsg")).Return(uint64(0), nil)
	client.On("NonceAt", mock.AnythingOfType("*context.emptyCtx"), mock.AnythingOfType("common.Address")).Return(uint64(1), nil)
	client.On("TransactionReceipt", mock.AnythingOfType("*context.emptyCtx"), mock.AnythingOfType("common.Hash")).Return(&types.Receipt{
		ContractAddress: common.HexToAddress(strings.Repeat("0", 20)),
	}, nil)

	client.On("SendTransaction", mock.AnythingOfType("*context.emptyCtx"), mock.MatchedBy(func(tx *types.Transaction) bool {
		return tx.Nonce() != 1
	})).Return(eth.SendTransactionResponse{}, stderr.New("Invalid transaction nonce"))

	client.On("SendTransaction", mock.AnythingOfType("*context.emptyCtx"), mock.MatchedBy(func(tx *types.Transaction) bool {
		return tx.Nonce() == 1
	})).Return(eth.SendTransactionResponse{
		Status: StatusOK,
		Output: "Success",
		Hash:   "Some hash",
	}, nil)
}

func initializeExecutor() (*TransactionExecutor, error) {
	privateKey, err := crypto.HexToECDSA(strings.Repeat("1", 64))
	if err != nil {
		return nil, fmt.Errorf("failed to initialize private key with error %s", err.Error())
	}

	logger := log.NewLogrus(log.LogrusLoggerProperties{})
	client := MockClient{}
	implementMockClient(&client)

	executor := NewTransactionExecutor(
		privateKey,
		types.FrontierSigner{},
		0,
		&client,
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
	executor.client.(*MockClient).AssertNumberOfCalls(t, "SendTransaction", 2)
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
	executor.client.(*MockClient).AssertNumberOfCalls(t, "SendTransaction", 2)
}
