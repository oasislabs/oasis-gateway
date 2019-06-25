package ethtest

import (
	"context"
	"math/big"

	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/oasislabs/developer-gateway/eth"
	"github.com/stretchr/testify/mock"
)

type MockClient struct {
	mock.Mock
}

func (c *MockClient) BalanceAt(
	ctx context.Context,
	address common.Address,
	block *big.Int,
) (*big.Int, error) {
	args := c.Called(ctx, address, block)
	if args.Get(1) != nil {
		return nil, args.Error(1)
	}

	return args.Get(0).(*big.Int), nil
}

func (m *MockClient) EstimateGas(
	ctx context.Context,
	msg ethereum.CallMsg,
) (uint64, error) {
	args := m.Called(ctx, msg)
	return args.Get(0).(uint64), args.Error(1)
}

func (m *MockClient) GetPublicKey(
	ctx context.Context,
	addr common.Address,
) (eth.PublicKey, error) {
	args := m.Called(ctx, addr)
	return args.Get(0).(eth.PublicKey), args.Error(1)
}

func (m *MockClient) NonceAt(
	ctx context.Context,
	addr common.Address,
) (uint64, error) {
	args := m.Called(ctx, addr)
	return args.Get(0).(uint64), args.Error(1)
}

func (m *MockClient) GetCode(
	ctx context.Context,
	addr common.Address,
) ([]byte, error) {
	args := m.Called(ctx, addr)
	return args.Get(0).([]byte), args.Error(1)
}

func (m *MockClient) SendTransaction(
	ctx context.Context,
	tx *types.Transaction,
) (eth.SendTransactionResponse, error) {
	args := m.Called(ctx, tx)
	return args.Get(0).(eth.SendTransactionResponse), args.Error(1)
}

func (m *MockClient) SubscribeFilterLogs(
	ctx context.Context,
	q ethereum.FilterQuery,
	c chan<- types.Log,
) (ethereum.Subscription, error) {
	args := m.Called(ctx, q, c)
	return args.Get(0).(*MockSubscription), args.Error(1)
}

func (m *MockClient) TransactionReceipt(ctx context.Context, txHash common.Hash) (*types.Receipt, error) {
	args := m.Called(ctx, txHash)
	return args.Get(0).(*types.Receipt), args.Error(1)
}
