package core

import (
	"context"
	"crypto/ecdsa"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"

	"github.com/oasislabs/developer-gateway/errors"
	"github.com/oasislabs/developer-gateway/eth"
	"github.com/oasislabs/developer-gateway/log"
)

// Wallet is an interface for any type that signs transactions
// and receives responses
type Wallet interface {
	Address() common.Address
	TransactionClient() eth.Client
	TransactionNonce() uint64
	UpdateNonce(ctx context.Context) errors.Err
	SignTransaction(tx *types.Transaction) (*types.Transaction, errors.Err)
}

type InternalWallet struct {
	privateKey *ecdsa.PrivateKey
	signer     types.Signer
	nonce      uint64
	client     eth.Client
	logger     log.Logger
}

func NewWallet(
	privateKey *ecdsa.PrivateKey,
	signer types.Signer,
	nonce uint64,
	client eth.Client,
	logger log.Logger,
) *InternalWallet {
	w := &InternalWallet{
		privateKey: privateKey,
		signer:     signer,
		nonce:      nonce,
		client:     client,
		logger:     logger,
	}

	return w
}

func (w *InternalWallet) Address() common.Address {
	return crypto.PubkeyToAddress(w.privateKey.PublicKey)
}

func (w *InternalWallet) TransactionClient() eth.Client {
	return w.client
}

func (w *InternalWallet) TransactionNonce() uint64 {
	nonce := w.nonce
	w.nonce++
	return nonce
}

func (w *InternalWallet) UpdateNonce(ctx context.Context) errors.Err {
	var err error
	for attempts := 0; attempts < 10; attempts++ {

		address := w.Address().Hex()
		nonce, err := w.client.NonceAt(ctx, common.HexToAddress(address))
		if err != nil {
			w.logger.Debug(ctx, "NonceAt request failed", log.MapFields{
				"call_type": "NonceFailure",
				"address":   address,
			}, errors.New(errors.ErrFetchNonce, err))
			continue
		}

		if w.nonce < nonce {
			w.nonce = nonce

			w.logger.Debug(ctx, "", log.MapFields{
				"call_type": "NonceSuccess",
				"address":   address,
			})

			return nil
		}
	}

	w.logger.Debug(ctx, "Exceeded NonceAt request limit", log.MapFields{
		"call_type": "NonceFailure",
	}, errors.New(errors.ErrFetchNonce, err))

	return errors.New(errors.ErrFetchNonce, err)
}

func (w *InternalWallet) SignTransaction(tx *types.Transaction) (*types.Transaction, errors.Err) {
	var err error
	tx, err = types.SignTx(tx, w.signer, w.privateKey)
	if err != nil {
		err := errors.New(errors.ErrSignedTx, err)
		return nil, err
	}

	return tx, nil
}
