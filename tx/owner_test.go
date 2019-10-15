package tx

import (
	"context"
	"math/big"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"

	"github.com/oasislabs/oasis-gateway/callback/callbacktest"
	callback "github.com/oasislabs/oasis-gateway/callback/client"
	"github.com/oasislabs/oasis-gateway/eth"
	"github.com/oasislabs/oasis-gateway/eth/ethtest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

const address string = "0x6f6704e5a10332af6672e50b3d9754dc460dfa4d"

func mockClientForNonce(client *ethtest.MockClient) {
	client.On("EstimateGas",
		mock.AnythingOfType("*context.emptyCtx"),
		mock.AnythingOfType("ethereum.CallMsg")).
		Return(uint64(0), nil)
	client.On("NonceAt",
		mock.AnythingOfType("*context.emptyCtx"),
		mock.AnythingOfType("common.Address")).
		Return(uint64(1), nil)
	client.On("GetCode",
		mock.AnythingOfType("*context.emptyCtx"),
		mock.AnythingOfType("common.Address")).
		Return("0x0000000000000000000000000000000000000000", nil)
	client.On("BalanceAt",
		mock.AnythingOfType("*context.emptyCtx"),
		mock.AnythingOfType("common.Address"),
		mock.AnythingOfType("*big.Int")).
		Return(big.NewInt(1), nil)
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
		Return(eth.SendTransactionResponse{}, eth.ErrInvalidNonce)
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

func mockClientForWalletOutOfFundsBodyCallback(client *ethtest.MockClient) {
	client.On("EstimateGas",
		mock.AnythingOfType("*context.emptyCtx"),
		mock.AnythingOfType("ethereum.CallMsg")).
		Return(uint64(0), nil)
	client.On("NonceAt",
		mock.AnythingOfType("*context.emptyCtx"),
		mock.AnythingOfType("common.Address")).
		Return(uint64(1), nil)
	client.On("BalanceAt",
		mock.AnythingOfType("*context.emptyCtx"),
		mock.AnythingOfType("common.Address"),
		mock.AnythingOfType("*big.Int")).
		Return(big.NewInt(1), nil)
	client.On("TransactionReceipt",
		mock.AnythingOfType("*context.emptyCtx"),
		mock.AnythingOfType("common.Hash")).
		Return(&types.Receipt{
			ContractAddress: common.HexToAddress(strings.Repeat("0", 20)),
		}, nil)
	client.On("SendTransaction",
		mock.AnythingOfType("*context.emptyCtx"),
		mock.Anything).
		Return(eth.SendTransactionResponse{}, eth.ErrExceedsBalance)
}

func newOwner(client *ethtest.MockClient) (*WalletOwner, error) {
	callbackclient := &callbacktest.MockClient{}
	callbacktest.ImplementMock(callbackclient)
	return NewWalletOwner(
		context.TODO(),
		&WalletOwnerServices{
			Client:    client,
			Callbacks: callbackclient,
			Logger:    Logger,
		},
		&WalletOwnerProps{
			PrivateKey: GetPrivateKey(),
			Signer:     types.FrontierSigner{},
			Nonce:      0,
		})
}

func TestTransactionNonce(t *testing.T) {
	mockclient := &ethtest.MockClient{}
	mockClientForNonce(mockclient)
	owner, err := newOwner(mockclient)
	assert.Nil(t, err)

	var nonce uint64
	for i := 0; i < 10; i++ {
		nonce = owner.transactionNonce()
		assert.Equal(t, uint64(i+1), nonce)
	}
}

func TestExecutorSignTransaction(t *testing.T) {
	mockclient := &ethtest.MockClient{}
	mockClientForNonce(mockclient)
	owner, err := newOwner(mockclient)
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

	tx, err = owner.signTransaction(tx)
	assert.Nil(t, err)

	V, R, S := tx.RawSignatureValues()
	assert.NotEqual(t, new(big.Int), V)
	assert.NotEqual(t, new(big.Int), R)
	assert.NotEqual(t, new(big.Int), S)
}

func TestExecuteTransactionNoAddressBadNonce(t *testing.T) {
	mockclient := &ethtest.MockClient{}
	mockClientForNonce(mockclient)
	owner, err := newOwner(mockclient)
	assert.Nil(t, err)

	owner.nonce = 0
	_, err = owner.executeTransaction(context.TODO(), ExecuteRequest{
		ID:      0,
		Address: "",
		Data:    []byte(""),
	})

	assert.Nil(t, err)
	mockclient.AssertNumberOfCalls(t, "SendTransaction", 2)
}

func TestExecuteTransactionAddressBadNonce(t *testing.T) {
	mockclient := &ethtest.MockClient{}
	mockClientForNonce(mockclient)
	owner, err := newOwner(mockclient)
	assert.Nil(t, err)

	owner.nonce = 0
	_, err = owner.executeTransaction(context.TODO(), ExecuteRequest{
		ID:      0,
		Address: strings.Repeat("0", 20),
		Data:    []byte(""),
	})

	assert.Nil(t, err)
	mockclient.AssertNumberOfCalls(t, "SendTransaction", 2)
}

func TestExecuteTransactionExceedsBalance(t *testing.T) {
	mockclient := &ethtest.MockClient{}
	mockClientForWalletOutOfFundsBodyCallback(mockclient)
	owner, err := newOwner(mockclient)
	assert.Nil(t, err)
	mockcallback := owner.callbacks.(*callbacktest.MockClient)

	_, err = owner.executeTransaction(context.TODO(), ExecuteRequest{
		ID:      0,
		Address: strings.Repeat("0", 20),
		Data:    []byte(""),
	})

	assert.Error(t, err)

	mockcallback.AssertCalled(t, "WalletOutOfFunds", mock.Anything,
		mock.MatchedBy(func(body callback.WalletOutOfFundsBody) bool {
			return body.Address == owner.wallet.Address().Hex()
		}))
}

func TestOwnerWalletReachedFundsThresholdOnNewOK(t *testing.T) {
	mockclient := &ethtest.MockClient{}
	ethtest.ImplementMock(mockclient)
	owner, err := newOwner(mockclient)
	assert.Nil(t, err)
	mockcallback := owner.callbacks.(*callbacktest.MockClient)

	mockcallback.AssertCalled(t, "WalletReachedFundsThreshold", mock.Anything,
		mock.MatchedBy(func(body callback.WalletReachedFundsThresholdBody) bool {
			return body.Address == "0x0759BC19964B467FcadaFdA49BE7986CB27183E3" &&
				body.Before == nil &&
				body.After.Cmp(new(big.Int).SetInt64(1)) == 0
		}))
}

func TestWalletReachedFundsThresholdOnTransactionOK(t *testing.T) {
	mockclient := &ethtest.MockClient{}
	ethtest.ImplementMock(mockclient)
	owner, err := newOwner(mockclient)
	assert.Nil(t, err)

	// reset callbacks to test the call of a transaction
	callbackclient := &callbacktest.MockClient{}
	callbacktest.ImplementMock(callbackclient)
	owner.callbacks = callbackclient

	_, err = owner.executeTransaction(context.TODO(), ExecuteRequest{})

	assert.Nil(t, err)
	callbackclient.AssertCalled(t, "WalletReachedFundsThreshold", mock.Anything,
		mock.MatchedBy(func(body callback.WalletReachedFundsThresholdBody) bool {
			return body.Address == "0x0759BC19964B467FcadaFdA49BE7986CB27183E3" &&
				body.Before.Cmp(new(big.Int).SetInt64(1)) == 0 &&
				body.After.Cmp(new(big.Int).SetInt64(1)) == 0
		}))
}
