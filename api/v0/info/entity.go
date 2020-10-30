package info

// GetVersionResponse is the response to the health request.
type GetVersionResponse struct {
	Version int `json:"version"`
}

// GetEthSenders is the response to the GetSenders request.
type GetSendersResponse struct {
	// The set of (hex-encoded) Ethereum addresses that this gateway may use as the
	// signers of transactions.
	Addresses []string `json:"addresses"`
}
