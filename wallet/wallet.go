package wallet

import (
	"crypto/ecdsa"
	
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"

	"github.com/oasislabs/developer-gateway/errors"
)

// Wallet is an interface for any type that signs transactions
// and receives responses
type Wallet interface {
	Address() string
	SignTransaction(tx *types.Transaction, signer *types.Signer)
}

type InMemoryWallet struct {
	PrivateKey *ecdsa.PrivateKey
	Signer     types.Signer
}

func (w *InMemoryWallet) Address() common.Address {
	return crypto.PubkeyToAddress(w.PrivateKey.PublicKey)
}

func (w *InMemoryWallet) SignTransaction(tx *types.Transaction) (*types.Transaction, errors.Err) {
	var err error
	tx, err = types.SignTx(tx, w.Signer, w.PrivateKey)
	if err != nil {
		err := errors.New(errors.ErrSignedTx, err)
		return nil, err
	}

	return tx, nil
}
