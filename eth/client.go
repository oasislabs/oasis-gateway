package eth

import (
	"fmt"
)

type Wallet struct {
	PublicKey  string
	PrivateKey string
}

type EthClientProperties struct {
	Wallet Wallet
}

type EthClient struct {
	wallet Wallet
}

func (c *EthClient) Request(req interface{}) (interface{}, error) {
	return nil, fmt.Errorf("not implemented")
}

func NewEthClient(properties EthClientProperties) *EthClient {
	return &EthClient{
		wallet: properties.Wallet,
	}
}
