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
		wallet    string
		runtimeID uint64
	)
	pflag.StringVar(&wallet, "wallet", "", "the hex encoded private key of the wallet")
	pflag.Uint64Var(&runtimeID, "runtimeID", 0, "sets the runtime ID")
	pflag.Parse()

	if len(wallet) == 0 {
		fmt.Println("-wallet needs to be set")
		os.Exit(1)
	}

	privateKey, err := crypto.HexToECDSA(wallet)
	if err != nil {
		fmt.Println("failed to read private key with error ", err.Error())
		os.Exit(1)
	}

	ctx := context.Background()

	client, err := ekiden.DialContext(ctx, ekiden.ClientProps{
		RuntimeID:       runtimeIDToBytes(runtimeID),
		Wallet:          ekiden.Wallet{PrivateKey: privateKey},
		RuntimeProps:    ekiden.NodeProps{URL: "unix:///tmp/runtime-ethereum-single_node/internal.sock"},
		KeyManagerProps: ekiden.NodeProps{URL: "127.0.0.1:9003"},
	})
	if err != nil {
		fmt.Println("failed to dial ekiden client: ", err.Error())
		os.Exit(1)
	}

	r, err := client.GetPublicKey(ctx, core.GetPublicKeyRequest{
		Address: "f75d55dd51ee8756fbdb499cc1a963e702a52091",
	})
	fmt.Println("RES: ", r, err)
	// res, err := client.DeployService(ctx, 0, core.DeployServiceRequest{
	// 	Data: "0x608060405234801561001057600080fd5b506102dd806100206000396000f30060806040526004361061006d576000357c0100000000000000000000000000000000000000000000000000000000900463ffffffff1680635b34b966146100725780635caf51da146100895780637531dafc146100b65780638ada066e146100e3578063c847beca1461010e575b600080fd5b34801561007e57600080fd5b50610087610139565b005b34801561009557600080fd5b506100b460048036038101908080359060200190929190505050610184565b005b3480156100c257600080fd5b506100e160048036038101908080359060200190929190505050610226565b005b3480156100ef57600080fd5b506100f861028e565b6040518082815260200191505060405180910390f35b34801561011a57600080fd5b50610123610297565b6040518082815260200191505060405180910390f35b600160008082825401925050819055507f20d8a6f5a693f9d1d627a598e8820f7a55ee74c183aa8f1a30e8d4e8dd9a8d846000546040518082815260200191505060405180910390a1565b60005481141515610223576040517f08c379a00000000000000000000000000000000000000000000000000000000081526004018080602001828103825260288152602001807f636f756e74657220646f6573206e6f7420657175616c20746f2065787065637481526020017f65642076616c756500000000000000000000000000000000000000000000000081525060400191505060405180910390fd5b50565b60008090505b8181101561028a57600160008082825401925050819055507f20d8a6f5a693f9d1d627a598e8820f7a55ee74c183aa8f1a30e8d4e8dd9a8d846000546040518082815260200191505060405180910390a1808060010191505061022c565b5050565b60008054905090565b6000600160008082825401925050819055506000549050905600a165627a7a7230582050c3f65d85a5b90f18463eb18980ddea1cd3ed55e96cf8d267a7b394d3e33b9e0029",
	// 	Key:  "key",
	// })

	fmt.Println("RESULT: ", err)
}
