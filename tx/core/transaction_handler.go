package core

import (
	"context"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/oasislabs/developer-gateway/errors"
)

// SignRequest is the request to sign a transaction
type ExecuteRequest struct {
	// Key unique identifier of the wallet. If not specified, an available wallet is selected automatically.
	Key string

	// Transaction ID
	ID uint64

	// Address to which to execute transaction
	Address string

	// Transaction data
	Data []byte
}

// PublicKeyRequest is the request to retrieve the public key for a given address
type PublicKeyRequest struct {
	// Key unique identifier of the wallet. If not specified, an available wallet is selected automatically.
	Key string

	// Address from which to extract public key
	Address string
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
	// Execute a transaction
	Execute(context.Context, ExecuteRequest) (*types.Receipt, errors.Err)

	// Remove the wallet and associated resources with the key
	Remove(context.Context, RemoveRequest) errors.Err
}
