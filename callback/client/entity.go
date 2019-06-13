package client

// Callback is the definition of how a callback
// will be sent and what data along with it
type Callback struct {
	// Enabled if set the callback will be send by the
	// client, otherwise it will be ignored
	Enabled bool

	// Method is the http method send in the http request
	Method string

	// URL is the complete http url where the request will
	// be sent
	URL string

	// Body is the body of the http request that needs to
	// be sent
	Body string

	// Headers a slice of http headers (':' separated)
	// that will be sent through the client
	Headers []string
}

// WalletOutOfFundsBody is the body sent on a WalletOutOfFunds
// to the required endpoint.
type WalletOutOfFundsBody struct {
	// Address is the address of the wallet that is out of funds
	Address string
}
