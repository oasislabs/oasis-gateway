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
	conf *config.Config,
) (*eth.EthClient, error) {
	if len(conf.WalletConfig.PrivateKeys) == 0 {
		return nil, errors.New("private_key not set in configuration")
	}

	privateKeys := make([]*ecdsa.PrivateKey, len(conf.WalletConfig.PrivateKeys))
	for i := 0; i < len(conf.WalletConfig.PrivateKeys); i++ {
		privateKey, err := crypto.HexToECDSA(conf.WalletConfig.PrivateKeys[i])
		if err != nil {
			return nil, fmt.Errorf("failed to read private key with error %s", err.Error())
		}
		privateKeys[i] = privateKey
	}

	return eth.NewClient(ctx, gateway.RootLogger, privateKeys, EthMockClient{})
}

type EthMockClient struct{}

func (c EthMockClient) EstimateGas(ctx context.Context, msg ethereum.CallMsg) (uint64, error) {
	if hexutil.Encode(msg.Data) == TransactionDataErr {
		return 0, errors.New("failed transaction")
	}

	return 1234, nil
}

func (c EthMockClient) GetPublicKey(context.Context, common.Address) (ethimpl.PublicKey, error) {
	return ethimpl.PublicKey{
		Timestamp: 123456789097654321,
		PublicKey: "0x0000000000000000000000000000000000000000",
		Signature: "0x0000000000000000000000000000000000000000",
	}, nil
}

func (c EthMockClient) NonceAt(context.Context, common.Address) (uint64, error) {
	return 0, nil
}

func (c EthMockClient) SendTransaction(ctx context.Context, tx *types.Transaction) (ethimpl.SendTransactionResponse, error) {
	data := hexutil.Encode(tx.Data())

	switch {
	case data == TransactionDataErr:
		return ethimpl.SendTransactionResponse{}, errors.New("failed transaction")
	case data == TransactionDataReceiptErr:
		return ethimpl.SendTransactionResponse{
			Output: errorHex,
			Status: 0,
			Hash:   tx.Hash().Hex(),
		}, nil
	default:
		return ethimpl.SendTransactionResponse{
			Output: successHex,
			Status: 1,
			Hash:   tx.Hash().Hex(),
		}, nil
	}
}

func (c EthMockClient) SubscribeFilterLogs(
	context.Context,
	ethereum.FilterQuery,
	chan<- types.Log,
) (ethereum.Subscription, error) {
	return nil, errors.New("eth failure client error")
}

func (c EthMockClient) TransactionReceipt(
	ctx context.Context,
	txHash common.Hash,
) (*types.Receipt, error) {
	return &types.Receipt{TxHash: txHash, Status: 1}, nil
}
