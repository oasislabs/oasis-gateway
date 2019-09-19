package tx

// ExecuteRequest is the request to execute an Ethereum transaction
type ExecuteRequest struct {
	// AAD is the identifier of the original issuer for the transaction data
	AAD string

	// Transaction ID
	ID uint64

	// Address to which to execute transaction
	Address string

	// Transaction data
	Data []byte
}

type ExecuteResponse struct {
	Address string
	Output  string
	Hash    string
}
