package exec

import (
	"context"
	"crypto/ecdsa"
	"io/ioutil"
	"strconv"
	"strings"
	"testing"

	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/oasislabs/developer-gateway/eth"
	"github.com/oasislabs/developer-gateway/log"
	"github.com/oasislabs/developer-gateway/tx/core"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

var (
	ctx    = context.Background()
	logger = log.NewLogrus(log.LogrusLoggerProperties{
		Level:  logrus.DebugLevel,
		Output: ioutil.Discard,
	})
	numKeys = 2
)

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

func initializeServer() (*Server, context.CancelFunc) {
	pks := make([]*ecdsa.PrivateKey, numKeys)
	for i := 0; i < numKeys; i++ {
		privateKey, _ := crypto.HexToECDSA(strings.Repeat(strconv.Itoa(i+1), 64))
		pks[i] = privateKey
	}
	ctx, cancel := context.WithCancel(context.Background())
	s, err := NewServer(ctx, &ServerServices{
		Logger: logger,
		Client: &MockClient{},
	}, &ServerProps{PrivateKeys: pks})

	if err != nil {
		return nil, cancel
	}
	return s, cancel
}

func TestServerRemove(t *testing.T) {
	s, cancel := initializeServer()
	defer cancel()

	pk, _ := crypto.HexToECDSA(strings.Repeat("1", 64))

	err := s.Remove(ctx, core.RemoveRequest{Key: crypto.PubkeyToAddress(pk.PublicKey).Hex()})
	assert.Nil(t, err)
}
