package tx

// ExecuteRequest is the request to execute an Ethereum transaction
type ExecuteRequest struct {
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
