package main

import (
	"context"
	"encoding/binary"
	"fmt"
	"os"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/oasislabs/developer-gateway/backend/core"
	"github.com/oasislabs/developer-gateway/backend/ekiden"
	"github.com/spf13/pflag"
)

func runtimeIDToBytes(runtimeID uint64) []byte {
	p := make([]byte, 32)
	binary.LittleEndian.PutUint64(p, runtimeID)
	return p
}

func main() {
	var (
		wallet       string
		registryURL  string
		schedulerURL string
		runtimeID    uint64
	)
	pflag.StringVar(&wallet, "wallet", "", "the hex encoded private key of the wallet")
	pflag.StringVar(&registryURL, "registryURL", "", "sets the URL for the registry node")
	pflag.StringVar(&schedulerURL, "schedulerURL", "", "sets the URL for the scheduler node")
	pflag.Uint64Var(&runtimeID, "runtimeID", 0, "sets the runtime ID")
	pflag.Parse()

	if len(wallet) == 0 {
		fmt.Println("-wallet needs to be set")
		os.Exit(1)
	}

	if len(registryURL) == 0 {
		fmt.Println("-registryURL needs to be set")
		os.Exit(1)
	}

	if len(schedulerURL) == 0 {
		fmt.Println("-schedulerURL needs to be set")
		os.Exit(1)
	}

	privateKey, err := crypto.HexToECDSA(wallet)
	if err != nil {
		fmt.Sprintf("failed to read private key with error %s", err.Error())
		os.Exit(1)
	}

	ctx := context.Background()
	discovery, err := ekiden.DialContext(ctx, ekiden.DiscoveryProps{
		RuntimeID: runtimeIDToBytes(runtimeID),
		Registry: ekiden.NodeProps{
			URL:       registryURL,
			TLSConfig: nil,
		},
		Scheduler: ekiden.NodeProps{
			URL:       schedulerURL,
			TLSConfig: nil,
		},
	})

	if err != nil {
		fmt.Println("failed to dial ekiden discovery: ", err.Error())
		os.Exit(1)
	}

	client := ekiden.NewClient(discovery, ekiden.ClientProps{
		Wallet: ekiden.Wallet{PrivateKey: privateKey},
	})
	res, err := client.ExecuteService(ctx, 0, core.ExecuteServiceRequest{
		Data:    "some data",
		Address: "address",
		Key:     "key",
	})

	fmt.Println("RESULT: ", res, err)
}
