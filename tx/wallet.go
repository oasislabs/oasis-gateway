package tx

import (
	"crypto/ecdsa"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"

	"github.com/oasislabs/oasis-gateway/errors"
)

// Wallet is an interface for any type that signs transactions
// and receives responses
type Wallet interface {
	Address() common.Address
	SignTransaction(tx *types.Transaction) (*types.Transaction, errors.Err)
}

type InternalWallet struct {
	privateKey *ecdsa.PrivateKey
	signer     types.Signer
}

func NewWallet(
	privateKey *ecdsa.PrivateKey,
	signer types.Signer,
) *InternalWallet {
	w := &InternalWallet{
		privateKey: privateKey,
		signer:     signer,
	}

	return w
}

func (w *InternalWallet) Address() common.Address {
	return crypto.PubkeyToAddress(w.privateKey.PublicKey)
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
