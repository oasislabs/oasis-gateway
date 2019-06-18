package main

import (
	"context"
	"crypto/ecdsa"
	"fmt"

	"github.com/ethereum/go-ethereum/crypto"

	"github.com/oasislabs/developer-gateway/backend/eth"
	ethereum "github.com/oasislabs/developer-gateway/eth"
	"github.com/oasislabs/developer-gateway/log"
	"github.com/oasislabs/developer-gateway/tx"
)

type ClientProps struct {
	PrivateKey string
	URL        string
}

func dialClient(props ClientProps) (*eth.Client, error) {
	privateKey, err := crypto.HexToECDSA(props.PrivateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to parse private key with error %s", err.Error())
	}

	ctx := context.Background()
	logger := log.NewLogrus(log.LogrusLoggerProperties{})
	ethClient, err := ethereum.NewClient(ctx, &ethereum.Config{
		URL: props.URL,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to endpoint %s", err.Error())
	}

	executor, err := tx.NewExecutor(ctx, &tx.Deps{
		Logger: logger,
		Client: ethClient,
	}, &tx.Props{
		PrivateKeys: []*ecdsa.PrivateKey{privateKey},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create tx executor %s", err.Error())
	}

	client := eth.NewClient(ctx, &eth.Deps{
		Logger:   logger,
		Executor: executor,
		Client:   ethClient,
	})

	return client, nil
}
