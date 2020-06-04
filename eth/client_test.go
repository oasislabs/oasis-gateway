package eth

import (
	"context"
	"errors"
	"math/big"
	"testing"
	"time"

	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/oasislabs/oasis-gateway/concurrent"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

var TestRetryConfig = concurrent.RetryConfig{
	BaseTimeout:     1,
	BaseExp:         1,
	MaxRetryTimeout: 10 * time.Millisecond,
	Attempts:        10,
	Random:          false,
}

const privateKey string = "a0ae0f77853ea56afe555133530f0960d4a6a3245b129ffde8d1d0cb35cc6bfc"

func getSignedTransaction() (*types.Transaction, error) {
	pk, err := crypto.HexToECDSA(privateKey)
	if err != nil {
		return nil, err
	}

	tx := types.NewTransaction(0,
		common.Address([20]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}),
		big.NewInt(0), 1024, big.NewInt(1), []byte("0x00"))
	signer := types.FrontierSigner{}

	return types.SignTx(tx, signer, pk)
}

type mockEthClient struct {
	mock.Mock
}

func (c *mockEthClient) BalanceAt(ctx context.Context, address common.Address, block *big.Int) (*big.Int, error) {
	args := c.Called(ctx, address, block)
	if args.Get(1) != nil {
		return nil, args.Error(1)
	}

	return args.Get(0).(*big.Int), nil
}

func (c *mockEthClient) CodeAt(ctx context.Context, address common.Address, block *big.Int) ([]byte, error) {
	args := c.Called(ctx, address, block)
	if args.Get(1) != nil {
		return nil, args.Error(1)
	}

	return args.Get(0).([]byte), nil
}

func (c *mockEthClient) EstimateGas(ctx context.Context, msg ethereum.CallMsg) (uint64, error) {
	args := c.Called(ctx, msg)
	if args.Get(1) != nil {
		return 0, args.Error(1)
	}

	return args.Get(0).(uint64), nil
}

func (c *mockEthClient) NonceAt(ctx context.Context, account common.Address, n *big.Int) (uint64, error) {
	args := c.Called(ctx, account, n)
	if args.Get(1) != nil {
		return 0, args.Error(1)
	}

	return args.Get(0).(uint64), nil
}

func (c *mockEthClient) TransactionReceipt(ctx context.Context, txHash common.Hash) (*types.Receipt, error) {
	args := c.Called(ctx, txHash)
	if args.Get(1) != nil {
		return nil, args.Error(1)
	}

	return args.Get(0).(*types.Receipt), nil
}

func (c *mockEthClient) SubscribeFilterLogs(ctx context.Context, q ethereum.FilterQuery, ch chan<- types.Log) (ethereum.Subscription, error) {
	args := c.Called(ctx, q, ch)
	if args.Get(1) != nil {
		return nil, args.Error(1)
	}

	return args.Get(0).(ethereum.Subscription), nil
}

func (c *mockEthClient) Close() {
	c.Called()
}

type mockRpcClient struct {
	mock.Mock
}

func (c *mockRpcClient) CallContext(ctx context.Context, v interface{}, method string, params ...interface{}) error {
	args := c.Called(ctx, v, method, params)
	return args.Error(0)
}

func (c *mockRpcClient) Close() {
	c.Called()
}

type mockPool struct {
	conn *Conn
}

func (p mockPool) Conn(context.Context) (*Conn, error) {
	return p.conn, nil
}

func (p mockPool) Report(context.Context, *Conn) error {
	return nil
}

func TestPooledClientSendTransactionOK(t *testing.T) {
	pool := mockPool{conn: &Conn{eclient: &mockEthClient{}, rclient: &mockRpcClient{}}}
	c := NewPooledClient(PooledClientProps{
		Pool:        pool,
		RetryConfig: TestRetryConfig,
	})

	tx, err := getSignedTransaction()
	assert.Nil(t, err)
	data, err := rlp.EncodeToBytes(tx)
	assert.Nil(t, err)

	pool.conn.rclient.(*mockRpcClient).
		On("CallContext", mock.Anything, mock.Anything, "oasis_invoke", []interface{}{hexutil.Encode(data)}).
		Run(func(args mock.Arguments) {
			res := args[1].(*sendTransactionResponseDeserialize)
			res.Hash = tx.Hash().Hex()
			res.Output = "0x00"
			res.Status = "0x1"
		}).
		Return(nil)

	res, err := c.SendTransaction(context.Background(), tx)
	assert.Nil(t, err)
	assert.Equal(t, SendTransactionResponse{
		Output: "0x00",
		Status: 1,
		Hash:   tx.Hash().Hex(),
	}, res)
}

func TestPooledClientSendTransactionStatusErr(t *testing.T) {
	pool := mockPool{conn: &Conn{eclient: &mockEthClient{}, rclient: &mockRpcClient{}}}
	c := NewPooledClient(PooledClientProps{
		Pool:        pool,
		RetryConfig: TestRetryConfig,
	})

	tx, err := getSignedTransaction()
	assert.Nil(t, err)
	data, err := rlp.EncodeToBytes(tx)
	assert.Nil(t, err)

	pool.conn.rclient.(*mockRpcClient).
		On("CallContext", mock.Anything, mock.Anything, "oasis_invoke", []interface{}{hexutil.Encode(data)}).
		Run(func(args mock.Arguments) {
			res := args[1].(*sendTransactionResponseDeserialize)
			res.Hash = tx.Hash().Hex()
			res.Output = "0x00"
			res.Status = "0x0"
		}).
		Return(nil)

	res, err := c.SendTransaction(context.Background(), tx)
	assert.Nil(t, err)
	assert.Equal(t, SendTransactionResponse{
		Output: "0x00",
		Status: 0,
		Hash:   tx.Hash().Hex(),
	}, res)
}

func TestPooledClientSendTransactionCallErr(t *testing.T) {
	pool := mockPool{conn: &Conn{eclient: &mockEthClient{}, rclient: &mockRpcClient{}}}
	c := NewPooledClient(PooledClientProps{
		Pool:        pool,
		RetryConfig: TestRetryConfig,
	})

	tx, err := getSignedTransaction()
	assert.Nil(t, err)

	pool.conn.rclient.(*mockRpcClient).
		On("CallContext", mock.Anything, mock.Anything, "oasis_invoke", mock.Anything).
		Return(errors.New("error"))

	_, err = c.SendTransaction(context.Background(), tx)
	assert.Error(t, err)
	assert.Equal(t, "maximum number of attempts 10 reached; see cause for last error: error", err.Error())
}
