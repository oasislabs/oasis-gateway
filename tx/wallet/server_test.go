package wallet

import (
	"context"
	"crypto/ecdsa"
	"io/ioutil"
	"math/big"
	"strconv"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/oasislabs/developer-gateway/eth"
	"github.com/oasislabs/developer-gateway/log"
	"github.com/oasislabs/developer-gateway/tx/core"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

var (
	ctx     = context.Background()
	logger  = log.NewLogrus(log.LogrusLoggerProperties{
		Level:  logrus.DebugLevel,
		Output: ioutil.Discard,
	})
	numKeys = 2
)

func initializeServer() (*Server, context.CancelFunc) {
	dialer := eth.NewUniDialer(ctx, "https://localhost:1111")
	pks := make([]*ecdsa.PrivateKey, numKeys)
	for i := 0; i < numKeys; i++ {
		privateKey, _ := crypto.HexToECDSA(strings.Repeat(strconv.Itoa(i+1), 64))
		pks[i] = privateKey
	}
	ctx, cancel := context.WithCancel(context.Background())
	s, err := NewServer(ctx, logger, pks, dialer)

	if err != nil {
		return nil, cancel
	}
	return s, cancel
}

func TestServerSignBasic(t *testing.T) {
	s, cancel := initializeServer()
	defer cancel()

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

	tx, err := s.Sign(ctx, core.SignRequest{
		Transaction: tx,
	})
	assert.Nil(t, err)

	V, R, S := tx.RawSignatureValues()
	assert.NotEqual(t, new(big.Int), V)
	assert.NotEqual(t, new(big.Int), R)
	assert.NotEqual(t, new(big.Int), S)
}

func TestServerSignWithDifferentWallets(t *testing.T) {
	s, cancel := initializeServer()
	defer cancel()

	pk1, _ := crypto.HexToECDSA(strings.Repeat("1", 64))
	pk2, _ := crypto.HexToECDSA(strings.Repeat("2", 64))

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

	tx1, err := s.Sign(ctx, core.SignRequest{
		Key:         crypto.PubkeyToAddress(pk1.PublicKey).Hex(),
		Transaction: tx,
	})
	assert.Nil(t, err)

	V1, R1, S1 := tx1.RawSignatureValues()
	assert.NotEqual(t, new(big.Int), V1)
	assert.NotEqual(t, new(big.Int), R1)
	assert.NotEqual(t, new(big.Int), S1)

	tx2, err := s.Sign(ctx, core.SignRequest{
		Key:         crypto.PubkeyToAddress(pk2.PublicKey).Hex(),
		Transaction: tx,
	})
	assert.Nil(t, err)

	V2, R2, S2 := tx2.RawSignatureValues()
	assert.NotEqual(t, new(big.Int), V2)
	assert.NotEqual(t, new(big.Int), R2)
	assert.NotEqual(t, new(big.Int), S2)

	// Assert different signature for different signing keys
	assert.NotEqual(t, R1, R2)
	assert.NotEqual(t, S1, S2)
}

func TestServerRemove(t *testing.T) {
	s, cancel := initializeServer()
	defer cancel()

	pk, _ := crypto.HexToECDSA(strings.Repeat("1", 64))

	err := s.Remove(ctx, core.RemoveRequest{Key: crypto.PubkeyToAddress(pk.PublicKey).Hex()})
	assert.Nil(t, err)
}
