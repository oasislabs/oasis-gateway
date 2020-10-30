package version

// GetVersionResponse is the response to the health request
type GetVersionResponse struct {
	Version int `json:"version"`
}

// GetEthSenders is the response to the signers request
type GetSendersResponse struct {
	// Hex-encoded Ethereum addresses.
	Senders []string `json:"senders"`
}
