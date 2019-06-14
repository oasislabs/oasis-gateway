package tx

import (
	"fmt"
	"math/big"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/assert"
)

func initializeWallet() (Wallet, error) {
	privateKey, err := crypto.HexToECDSA(strings.Repeat("1", 64))
	if err != nil {
		return nil, fmt.Errorf("failed to initialize private key with error %s", err.Error())
	}

	wallet := NewWallet(
		privateKey,
		types.FrontierSigner{},
	)

	return wallet, nil
}

func TestAddress(t *testing.T) {
	wallet, err := initializeWallet()
	assert.Nil(t, err)

	address := wallet.Address().Hex()
	assert.Equal(t, "0x19E7E376E7C213B7E7e7e46cc70A5dD086DAff2A", address)
}

func TestWalletSignTransaction(t *testing.T) {
	wallet, err := initializeWallet()
	assert.Nil(t, err)

	// Build a mock transaction
	gas := uint64(1000000)
	gasPrice := int64(1000000000)
	tx := types.NewTransaction(
		0,
		common.HexToAddress("0x6f6704e5a10332af6672e50b3d9754dc460dfa4d"),
		big.NewInt(0),
		gas,
		big.NewInt(gasPrice),
		[]byte("data"),
	)

	tx, err = wallet.SignTransaction(tx)
	assert.Nil(t, err)

	V, R, S := tx.RawSignatureValues()
	assert.NotEqual(t, new(big.Int), V)
	assert.NotEqual(t, new(big.Int), R)
	assert.NotEqual(t, new(big.Int), S)
}
