package tx

import (
	"context"
	stderr "errors"
	"math/big"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"

	"github.com/oasislabs/developer-gateway/eth"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

const address string = "0x6f6704e5a10332af6672e50b3d9754dc460dfa4d"

func mockClient(client *MockClient) {
	client.On("EstimateGas",
		mock.AnythingOfType("*context.emptyCtx"),
		mock.AnythingOfType("ethereum.CallMsg")).
		Return(uint64(0), nil)
	client.On("NonceAt",
		mock.AnythingOfType("*context.emptyCtx"),
		mock.AnythingOfType("common.Address")).
		Return(uint64(1), nil)
	client.On("TransactionReceipt",
		mock.AnythingOfType("*context.emptyCtx"),
		mock.AnythingOfType("common.Hash")).
		Return(&types.Receipt{
			ContractAddress: common.HexToAddress(strings.Repeat("0", 20)),
		}, nil)
	client.On("SendTransaction",
		mock.AnythingOfType("*context.emptyCtx"),
		mock.MatchedBy(func(tx *types.Transaction) bool {
			return tx.Nonce() == 0
		})).
		Return(eth.SendTransactionResponse{}, stderr.New("Invalid transaction nonce"))
	client.On("SendTransaction",
		mock.AnythingOfType("*context.emptyCtx"),
		mock.MatchedBy(func(tx *types.Transaction) bool {
			return tx.Nonce() == 1
		})).
		Return(eth.SendTransactionResponse{
			Status: StatusOK,
			Output: "Success",
			Hash:   "Some hash",
		}, nil)
}

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

func newOwner() *WalletOwner {
	return NewWalletOwner(
		&WalletOwnerServices{
			Client: &MockClient{},
			Logger: Logger,
		},
		&WalletOwnerProps{
			PrivateKey: GetPrivateKey(),
			Signer:     types.FrontierSigner{},
			Nonce:      0,
		})
}

func TestTransactionNonce(t *testing.T) {
	owner := newOwner()

	var nonce uint64
	for i := 0; i < 10; i++ {
		nonce = owner.transactionNonce()
		assert.Equal(t, uint64(i), nonce)
	}
}

func TestExecutorSignTransaction(t *testing.T) {
	owner := newOwner()

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

	tx, err := owner.signTransaction(tx)
	assert.Nil(t, err)

	V, R, S := tx.RawSignatureValues()
	assert.NotEqual(t, new(big.Int), V)
	assert.NotEqual(t, new(big.Int), R)
	assert.NotEqual(t, new(big.Int), S)
}

func TestExecuteTransactionNoAddressBadNonce(t *testing.T) {
	owner := newOwner()

	mockClient(owner.client.(*MockClient))

	req := executeRequest{
		ID:      0,
		Address: "",
		Data:    []byte(""),
	}
	_, err := owner.executeTransaction(context.TODO(), req)
	assert.Nil(t, err)
	owner.client.(*MockClient).AssertNumberOfCalls(t, "SendTransaction", 2)
}

func TestExecuteTransactionAddressBadNonce(t *testing.T) {
	owner := newOwner()

	mockClient(owner.client.(*MockClient))

	req := executeRequest{
		ID:      0,
		Address: strings.Repeat("0", 20),
		Data:    []byte(""),
	}

	_, err := owner.executeTransaction(context.TODO(), req)
	assert.Nil(t, err)
	owner.client.(*MockClient).AssertNumberOfCalls(t, "SendTransaction", 2)
}
