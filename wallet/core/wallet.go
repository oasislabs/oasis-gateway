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
	PrivateKey *ecdsa.PrivateKey
	Signer     types.Signer
	Nonce			 uint64
	Client     eth.Client
	Logger     log.Logger
}

func (w *InternalWallet) Address() common.Address {
	return crypto.PubkeyToAddress(w.PrivateKey.PublicKey)
}

func (w *InternalWallet) TransactionClient() eth.Client {
	return w.Client
}

func (w *InternalWallet) TransactionNonce() uint64 {
	nonce := w.Nonce
	w.Nonce++
	return nonce
}

func (w *InternalWallet) UpdateNonce(ctx context.Context) errors.Err {
	var err error
	for attempts := 0; attempts < 10; attempts++ {

		address := w.Address().Hex()
		nonce, err := w.Client.NonceAt(ctx, common.HexToAddress(address))
		if err != nil {
			w.Logger.Debug(ctx, "NonceAt request failed", log.MapFields{
				"call_type": "NonceFailure",
				"address":   address,
			}, errors.New(errors.ErrFetchNonce, err))
			continue
		}

		if w.Nonce < nonce {
			w.Nonce = nonce

			w.Logger.Debug(ctx, "", log.MapFields{
				"call_type": "NonceSuccess",
				"address":   address,
			})

			return nil
		}
	}

	w.Logger.Debug(ctx, "Exceeded NonceAt request limit", log.MapFields{
		"call_type": "NonceFailure",
	}, errors.New(errors.ErrFetchNonce, err))

	return errors.New(errors.ErrFetchNonce, err)
}

func (w *InternalWallet) SignTransaction(tx *types.Transaction) (*types.Transaction, errors.Err) {
	var err error
	tx, err = types.SignTx(tx, w.Signer, w.PrivateKey)
	if err != nil {
		err := errors.New(errors.ErrSignedTx, err)
		return nil, err
	}

	return tx, nil
}
