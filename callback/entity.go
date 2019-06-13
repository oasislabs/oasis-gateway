package callback

// WalletOutOfFundsBody is the body sent on a WalletOutOfFunds
// to the required endpoint.
type WalletOutOfFundsBody struct {
	// Address is the address of the wallet that is out of funds
	Address string
}
