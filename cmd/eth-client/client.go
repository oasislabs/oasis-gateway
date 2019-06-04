package main

import (
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"

	"github.com/oasislabs/developer-gateway/backend/eth"
	"github.com/oasislabs/developer-gateway/conc"
	ethereum "github.com/oasislabs/developer-gateway/eth"
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

	ctx := context.Background()
	dialer := ethereum.NewUniDialer(ctx, props.URL)
	pooledClient := ethereum.NewPooledClient(ethereum.PooledClientProps{
		Pool:        dialer,
		RetryConfig: conc.RandomConfig,
	})
	logger := log.NewLogrus(log.LogrusLoggerProperties{})

	wallet := wallet.InternalWallet{
		PrivateKey: privateKey,
		Signer:     types.FrontierSigner{},
		Nonce:      0,
		Client:     pooledClient,
		Logger:     logger.ForClass("wallet", "InternalWallet"),
	}

	client, err := eth.DialContext(ctx, logger, eth.EthClientProperties{
		Wallet: wallet,
		URL:    props.URL,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to endpoint %s", err.Error())
	}

	return client, nil
}
