package ethtest

import (
	"context"
	"math/big"

	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/oasislabs/oasis-gateway/eth"
	"github.com/stretchr/testify/mock"
)

type MockMethod struct {
	Arguments []interface{}
	Return    []interface{}
	Run       func(mock.Arguments)
}

type MockMethods map[string]MockMethod

var DefaultMockMethods = map[string]MockMethod{
	"EstimateGas": {
		Arguments: []interface{}{mock.Anything, mock.Anything},
		Return:    []interface{}{uint64(0), nil},
	},
	"NonceAt": {
		Arguments: []interface{}{mock.Anything, mock.Anything},
		Return:    []interface{}{uint64(1), nil},
	},
	"GetCode": {
		Arguments: []interface{}{mock.Anything, mock.Anything},
		Return:    []interface{}{"0x0000000000000000000000000000000000000000", nil},
	},
	"BalanceAt": {
		Arguments: []interface{}{mock.Anything, mock.Anything, mock.Anything},
		Return:    []interface{}{big.NewInt(1), nil},
	},
	"TransactionReceipt": {
		Arguments: []interface{}{mock.Anything, mock.Anything},
		Return: []interface{}{
			&types.Receipt{
				Status:          1,
				ContractAddress: common.HexToAddress("0x0000000000000000000000000000000000000000"),
			}, nil,
		},
	},
	"GetExpiry": {
		Arguments: []interface{}{mock.Anything, mock.Anything},
		Return:    []interface{}{uint64(123456789), nil},
	},
	"GetPublicKey": {
		Arguments: []interface{}{mock.Anything, mock.Anything},
		Return: []interface{}{
			eth.PublicKey{
				Timestamp: 1234,
				PublicKey: "0x6f6704e5a10332af6672e50b3d9754dc460dfa4d",
				Signature: "0x6f6704e5a10332af6672e50b3d9754dc460dfa4d",
			}, nil,
		},
	},
	"SendTransaction": {
		Arguments: []interface{}{mock.Anything, mock.Anything},
		Return: []interface{}{
			eth.SendTransactionResponse{
				Status: 1,
				Output: "0x73756363657373",
				Hash:   "0x00000000000000000000000000000000000000000000000000000000000000000",
			}, nil,
		},
	},
	"SubscribeFilterLogs": {
		Arguments: []interface{}{mock.Anything, mock.Anything, mock.Anything},
		Return: []interface{}{
			&MockSubscription{ErrC: make(chan error)}, nil,
		},
	},
}

func OverwriteDefaults(overwrite MockMethods) MockMethods {
	methods := make(MockMethods)

	for key, value := range DefaultMockMethods {
		if o, ok := overwrite[key]; ok {
			methods[key] = o
		} else {
			methods[key] = value
		}
	}

	return methods
}

func ImplementMockWithOverwrite(client *MockClient, overwrite MockMethods) {
	ImplementMockWithMethods(client, OverwriteDefaults(overwrite))
}

func ImplementMockWithMethods(client *MockClient, methods MockMethods) {
	for key, method := range methods {
		call := client.On(key, method.Arguments...)
		if len(method.Return) > 0 {
			call = call.Return(method.Return...)
		}
		if method.Run != nil {
			_ = call.Run(method.Run)
		}
	}
}

func ImplementMock(client *MockClient) {
	ImplementMockWithMethods(client, DefaultMockMethods)
}

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

func (m *MockClient) GetExpiry(
	ctx context.Context,
	addr common.Address,
) (uint64, error) {
	args := m.Called(ctx, addr)
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
) (string, error) {
	args := m.Called(ctx, addr)
	return args.Get(0).(string), args.Error(1)
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
	if args.Get(1) != nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*MockSubscription), nil
}

func (m *MockClient) TransactionReceipt(ctx context.Context, txHash common.Hash) (*types.Receipt, error) {
	args := m.Called(ctx, txHash)
	return args.Get(0).(*types.Receipt), args.Error(1)
}
