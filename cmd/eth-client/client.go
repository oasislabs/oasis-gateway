package main

import (
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum/crypto"

	"github.com/oasislabs/developer-gateway/backend/eth"
	"github.com/oasislabs/developer-gateway/log"
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
	client, err := eth.DialContext(ctx, &eth.ClientServices{
		Logger: logger,
	}, &eth.ClientProps{
		PrivateKey: privateKey,
		URL:        props.URL,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to endpoint %s", err.Error())
	}

	return client, nil
}
