package ekiden

// SubmitRequest is the request to submit a transaction to
// ekiden
type SubmitRequest struct {
	// Method to be invoked for the request
	Method string

	// RuntimeID is the ID of the runtime that will handle the request
	RuntimeID []byte

	// Data is the RLP encoded representation of the data that
	// is sent
	Data []byte
}

// SubmitResponse is the runtime's response to a successfully
// processed request
type SubmitResponse struct {
	// Result contains the resulting value of a successful response
	Result interface{}
}

// EthereumTransactionRequest is the request to submit an ethereum
// transaction to ekiden
type EthereumTransactionRequest struct {
	// RuntimeID is the ID of the runtime that will handle the request
	RuntimeID []byte

	// Data is the RLP encoded representation of the data that
	// is sent
	Data []byte
}

// EthereumTransactionResponse is the runtime's response to a successfully
// processed request
type EthereumTransactionResponse struct {
	// Result contains the resulting value of a successful response
	Result interface{}
}

// GetCodeRequest is a request from a client to retrieve the
// source code associated with a specific service
type GetCodeRequest struct {
	Address Address
}

// GetCodeResponse contains the source code associated with the address
type GetCodeResponse struct {
	Payload []byte
}

// GetPublicKeyRequest is a request from a client to retrieve the
// public key associated with a specific service
type GetPublicKeyRequest struct {
	Address Address
}

// GetPublicKeyResponse contains the public key associated with the
// address along with the expiration time
type GetPublicKeyResponse struct {
	Payload []byte
}

// CallEnclaveRequest
type CallEnclaveRequest struct {
	// Method to be invoked by the request
	Method string

	// Data is the RLP encoded representation of the data that
	// is sent
	Data []byte
}

// CallEnclaveResponse
type CallEnclaveResponse struct {
	// Result contains the resulting value of a successful response
	Payload interface{}
}
