package eth

type PublicKey struct {
	Timestamp uint64 `json:"timestamp"`
	PublicKey string `json:"public_key"`
	Signature string `json:"signature"`
}

type SendTransactionResponse struct {
	Output string `json:"output"`
	Status uint64 `json:"status"`
	Hash   string `json:"transactionHash"`
}

type sendTransactionResponseDeserialize struct {
	Output string `json:"output"`
	Status string `json:"status"`
	Hash   string `json:"transactionHash"`
}
