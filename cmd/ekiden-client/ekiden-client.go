package main

import (
	"context"
	"crypto/ecdsa"
	"encoding/binary"
	"fmt"
	"os"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/oasislabs/oasis-gateway/backend/core"
	"github.com/oasislabs/oasis-gateway/backend/ekiden"
	"github.com/oasislabs/oasis-gateway/log"
	"github.com/sirupsen/logrus"
	"github.com/spf13/pflag"
)

func runtimeIDToBytes(runtimeID uint64) []byte {
	p := make([]byte, 32)
	binary.LittleEndian.PutUint64(p, runtimeID)
	return p
}

func main() {
	var (
		walletKey string
		runtimeID uint64
	)
	pflag.StringVar(&walletKey, "walletKey", "", "the hex encoded private key of the wallet")
	pflag.Uint64Var(&runtimeID, "runtimeID", 0, "sets the runtime ID")
	pflag.Parse()

	if len(walletKey) == 0 {
		fmt.Println("-walletKey needs to be set")
		os.Exit(1)
	}

	privateKey, err := crypto.HexToECDSA(walletKey)
	if err != nil {
		fmt.Println("failed to read private key with error ", err.Error())
		os.Exit(1)
	}

	ctx := context.Background()

	client, err := ekiden.DialContext(ctx, ekiden.ClientProps{
		PrivateKeys:     []*ecdsa.PrivateKey{privateKey},
		RuntimeID:       runtimeIDToBytes(runtimeID),
		RuntimeProps:    ekiden.NodeProps{URL: "unix:///tmp/runtime-ethereum-single_node/internal.sock"},
		KeyManagerProps: ekiden.NodeProps{URL: "127.0.0.1:9003"},
		Logger: log.NewLogrus(log.LogrusLoggerProperties{
			Level: logrus.DebugLevel,
		}),
	})
	if err != nil {
		fmt.Println("failed to dial ekiden client: ", err.Error())
		os.Exit(1)
	}

	r, err := client.GetPublicKey(ctx, core.GetPublicKeyRequest{
		Address: "f75d55dd51ee8756fbdb499cc1a963e702a52091",
	})
	fmt.Println("RES: ", r, err)

	fmt.Println("RESULT: ", err)
}
