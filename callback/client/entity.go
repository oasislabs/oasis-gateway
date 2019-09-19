package client

import (
	"html/template"
	"math/big"
	"time"
)

// Callback is the definition of how a callback
// will be sent and what data along with it
type Callback struct {
	// Enabled if set the callback will be send by the
	// client, otherwise it will be ignored
	Enabled bool

	// Name is a human readable name to identify the callback
	Name string

	// Method is the http method send in the http request
	Method string

	// URL is the complete http url where the request will
	// be sent
	URL string

	// BodyFormat is the body of the http request that needs to
	// be sent
	BodyFormat *template.Template

	// QueryURLFormat is the query url of the http request that
	// will be sent in the callback
	QueryURLFormat *template.Template

	// Headers a slice of http headers (':' separated)
	// that will be sent through the client
	Headers []string

	// PeriodLimit is the minimum duration that there should
	// be between notifications of this callback
	PeriodLimit time.Duration

	// LastAttempt is the unix timestamp of the last time
	// the request type was attempted
	LastAttempt int64

	// Sync if true the callback will be sent synchronously
	Sync bool
}

// WalletOutOfFundsBody is the body sent on a WalletOutOfFunds
// callback to the required endpoint
type WalletOutOfFundsBody struct {
	// Address is the address of the wallet that is out of funds
	Address string
}

// WalletReachedFundsThresholdBody is the body sent on a WalletReachedFundsThresholdBody
// to the required endpoint
type WalletReachedFundsThresholdBody struct {
	// Address is the address of the wallet that reached the threshold
	Address string

	// Before the threshold of currency reached
	Before *big.Int

	// After the threshold of currency reached
	After *big.Int
}

// WalletReachedFundsThresholdRequest is the request sent on
// the callback
type WalletReachedFundsThresholdRequest struct {
	Address   string
	Before    string
	After     string
	Threshold string
}

// TransactionCommittedBody is the body sent on a TransactionCommitted
// callback to the required endpoint
type TransactionCommittedBody struct {
	// AAD is the unique identifier of the issuer of the data for the
	// transaction
	AAD string

	// Address is the wallet address that acted as a sender for the transaction
	Address string

	// Hash is the hash of the transaction that was committed
	Hash string
}
