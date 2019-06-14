package tx

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"io/ioutil"

	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/oasislabs/developer-gateway/eth"
	"github.com/oasislabs/developer-gateway/log"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/mock"
)

var (
	Logger = log.NewLogrus(log.LogrusLoggerProperties{
		Level:  logrus.DebugLevel,
		Output: ioutil.Discard,
	})
)

const (
	PrivateKey string = "17be884d0713e46a983fe65900c0ee0f45696cee60e5611ebc80841cfad407b7"
)

func GetPrivateKey() *ecdsa.PrivateKey {
	privateKey, err := crypto.HexToECDSA(PrivateKey)
	if err != nil {
		panic(fmt.Sprintf("failed to create private key: %s", err.Error()))
	}

	return privateKey
}

type MockClient struct {
	mock.Mock
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
	return args.Get(0).(ethereum.Subscription), args.Error(1)
}

func (m *MockClient) TransactionReceipt(ctx context.Context, txHash common.Hash) (*types.Receipt, error) {
	args := m.Called(ctx, txHash)
	return args.Get(0).(*types.Receipt), args.Error(1)
}
