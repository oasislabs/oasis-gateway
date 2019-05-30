package wallet

import (
	"context"
	"crypto/ecdsa"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"

	"github.com/oasislabs/developer-gateway/errors"
	"github.com/oasislabs/developer-gateway/eth"
	"github.com/oasislabs/developer-gateway/log"
)

type TransactionExecutor struct {
	wallet     *InternalWallet
	nonce      uint64
	client     eth.Client
	logger     log.Logger
}

func NewTransactionExecutor(
	privateKey *ecdsa.PrivateKey,
	signer types.Signer,
	nonce uint64,
	client eth.Client,
	logger log.Logger,
) *TransactionExecutor {
	wallet := NewWallet(privateKey, signer)
	executor := &TransactionExecutor{
		wallet:     wallet,
		nonce:      nonce,
		client:     client,
		logger:     logger,
	}

	return executor
}

func (e *TransactionExecutor) TransactionClient() eth.Client {
	return e.client
}

func (e *TransactionExecutor) TransactionNonce() uint64 {
	nonce := e.nonce
	e.nonce++
	return nonce
}

func (e *TransactionExecutor) UpdateNonce(ctx context.Context) errors.Err {
	var err error
	for attempts := 0; attempts < 10; attempts++ {

		address := e.wallet.Address().Hex()
		nonce, err := e.client.NonceAt(ctx, common.HexToAddress(address))
		if err != nil {
			e.logger.Debug(ctx, "NonceAt request failed", log.MapFields{
				"call_type": "NonceFailure",
				"address":   address,
			}, errors.New(errors.ErrFetchNonce, err))
			continue
		}

		if e.nonce < nonce {
			e.nonce = nonce

			e.logger.Debug(ctx, "", log.MapFields{
				"call_type": "NonceSuccess",
				"address":   address,
			})

			return nil
		}
	}

	e.logger.Debug(ctx, "Exceeded NonceAt request limit", log.MapFields{
		"call_type": "NonceFailure",
	}, errors.New(errors.ErrFetchNonce, err))

	return errors.New(errors.ErrFetchNonce, err)
}

func (e *TransactionExecutor) SignTransaction(tx *types.Transaction) (*types.Transaction, errors.Err) {
	return e.wallet.SignTransaction(tx)
}
