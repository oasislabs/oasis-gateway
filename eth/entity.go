package eth

type PublicKey struct {
	Timestamp uint64 `json:"timestamp"`
	PublicKey string `json:"public_key"`
	Signature string `json:"signature"`
}
