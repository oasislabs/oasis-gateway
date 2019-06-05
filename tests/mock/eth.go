package mock

import (
	"context"
	"crypto/ecdsa"
	"errors"
	"fmt"

	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/oasislabs/developer-gateway/backend/eth"
	ethimpl "github.com/oasislabs/developer-gateway/eth"
	"github.com/oasislabs/developer-gateway/gateway"
	"github.com/oasislabs/developer-gateway/gateway/config"
)

func NewMockEthClient(
	ctx context.Context,
	config config.Config,
) (*eth.EthClient, error) {
	if len(config.Wallet.PrivateKeys) == 0 {
		return nil, errors.New("private_keys not set in configuration")
	}

	privateKeys := make([]*ecdsa.PrivateKey, len(config.Wallet.PrivateKeys))
	for i := 0; i < len(config.Wallet.PrivateKeys); i++ {
		privateKey, err := crypto.HexToECDSA(config.Wallet.PrivateKeys[i])
		if err != nil {
			return nil, fmt.Errorf("failed to read private key with error %s", err.Error())
		}
		privateKeys[i] = privateKey
	}

	properties := eth.EthClientProperties{
		PrivateKeys: privateKeys,
		URL:         "",
	}
	return eth.NewClient(ctx, gateway.RootLogger, properties)
}

type EthFailureClient struct{}

func (c EthFailureClient) EstimateGas(ctx context.Context, msg ethereum.CallMsg) (uint64, error) {
	if hexutil.Encode(msg.Data) == TransactionDataErr {
		return 0, errors.New("failed transaction")
	}

	return 1234, nil
}

func (c EthFailureClient) GetPublicKey(context.Context, common.Address) (ethimpl.PublicKey, error) {
	return ethimpl.PublicKey{
		Timestamp: 123456789097654321,
		PublicKey: "0x0000000000000000000000000000000000000000",
		Signature: "0x0000000000000000000000000000000000000000",
	}, nil
}

func (c EthFailureClient) PendingNonceAt(context.Context, common.Address) (uint64, error) {
	return 0, errors.New("eth failure client error")
}

func (c EthFailureClient) SendTransaction(context.Context, *types.Transaction) error {
	return nil
}

func (c EthFailureClient) SubscribeFilterLogs(
	context.Context,
	ethereum.FilterQuery,
	chan<- types.Log,
) (ethereum.Subscription, error) {
	return nil, errors.New("eth failure client error")
}

func (c EthFailureClient) TransactionReceipt(
	ctx context.Context,
	txHash common.Hash,
) (*types.Receipt, error) {
	return &types.Receipt{TxHash: txHash, Status: 1}, nil
}
