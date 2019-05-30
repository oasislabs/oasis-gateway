package core

import (
	"context"
	"crypto/ecdsa"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/oasislabs/developer-gateway/errors"
)

// SignRequest is the request to sign a transaction
type SignRequest struct {
	// Key unique identifier of the wallet
	Key string

	// Transaction to be signed
	Transaction *types.Transaction
}

type GenerateRequest struct {
	// Key unique identifier of the wallet
	Key string

	// URL to which the generated wallet's ETH client should connect.
	URL string

	// Private key of the wallet
	PrivateKey *ecdsa.PrivateKey
}

// RemoveRequest to ask to destroy the wallet identified
// by the provided key
type RemoveRequest struct {
	// Key unique identifier of the wallet
	Key string
}

// TransactionHandler is an interface to a service that supports
// signing developer transactions.
type TransactionHandler interface {
	// Signs the provided transaction
	Sign(context.Context, SignRequest) (*types.Transaction, errors.Err)

	// Generates a new wallet to add to the wallet pool
	Generate(context.Context, GenerateRequest) errors.Err

	// Remove the wallet and associated resources with the key
	Remove(context.Context, RemoveRequest) errors.Err
}