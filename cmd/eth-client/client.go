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
	"github.com/oasislabs/developer-gateway/tx/wallet"
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

	handler := wallet.NewServer(ctx, logger)
	client, err := eth.DialContext(ctx, logger, eth.EthClientProperties{
		Handler: handler,
		URL:    props.URL,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to endpoint %s", err.Error())
	}

	return client, nil
}
