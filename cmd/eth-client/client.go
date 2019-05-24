package main

import (
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/oasislabs/developer-gateway/backend/eth"
	"github.com/oasislabs/developer-gateway/log"
	"github.com/oasislabs/developer-gateway/wallet"
)

type ClientProps struct {
	PrivateKey string
	URL        string
}

func dialClient(props ClientProps) (*eth.EthClient, error) {
	privateKey, err := crypto.HexToECDSA(props.PrivateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to parse private key with error %s", err.Error())
	}

	wallet := wallet.InMemoryWallet{
		PrivateKey: privateKey,
		Signer: types.FrontierSigner{},
	}
	ctx := context.Background()
	client, err := eth.DialContext(ctx, log.NewLogrus(log.LogrusLoggerProperties{}), eth.EthClientProperties{
		Wallet: wallet,
		URL:    props.URL,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to endpoint %s", err.Error())
	}

	return client, nil
}
