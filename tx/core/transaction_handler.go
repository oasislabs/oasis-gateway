package core

import (
	"context"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/oasislabs/developer-gateway/errors"
)

// SignRequest is a request to sign a generic transaction
type SignRequest struct {
	// Transaction
	Transaction *types.Transaction
}

// ExecuteRequest is the request to execute an Ethereum transaction
type ExecuteRequest struct {
	// Transaction ID
	ID uint64

	// Address to which to execute transaction
	Address string

	// Transaction data
	Data []byte
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
	// Sign a transaction
	Sign(context.Context, SignRequest) (*types.Transaction, errors.Err)

	// Execute a transaction
	Execute(context.Context, ExecuteRequest) (*types.Receipt, errors.Err)

	// Remove the wallet and associated resources with the key
	Remove(context.Context, RemoveRequest) errors.Err
}
