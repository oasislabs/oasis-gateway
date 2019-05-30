package wallet

import (
	"context"
	"io/ioutil"
	"math/big"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/oasislabs/developer-gateway/conc"
	"github.com/oasislabs/developer-gateway/eth"
	"github.com/oasislabs/developer-gateway/log"
	"github.com/oasislabs/developer-gateway/tx/core"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

var (
	ctx    = context.Background()
	logger = log.NewLogrus(log.LogrusLoggerProperties{
		Level:  logrus.DebugLevel,
		Output: ioutil.Discard,
	})
)

func initializeServer() (*Server, context.CancelFunc) {
	dialer := eth.NewUniDialer(ctx, "https://localhost:1111")
	client := eth.NewPooledClient(eth.PooledClientProps{
		Pool:        dialer,
		RetryConfig: conc.RandomConfig,
	})
	ctx, cancel := context.WithCancel(context.Background())
	s := NewServer(ctx, logger, client)

	return s, cancel
}

func TestServerGenerate(t *testing.T) {
	s, cancel := initializeServer()
	defer cancel()

	privateKey, err := crypto.HexToECDSA(strings.Repeat("1", 64))
	assert.Nil(t, err)

	err = s.Generate(ctx, core.GenerateRequest{
		Key:        "key",
		PrivateKey: privateKey,
	})
	assert.Nil(t, err)
}

func TestServerSign(t *testing.T) {
	s, cancel := initializeServer()
	defer cancel()

	// Generate a wallet
	privateKey, err := crypto.HexToECDSA(strings.Repeat("1", 64))
	assert.Nil(t, err)

	err = s.Generate(ctx, core.GenerateRequest{
		Key:        "key",
		PrivateKey: privateKey,
	})
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

	tx, err = s.Sign(ctx, core.SignRequest{
		Key:         "key",
		Transaction: tx,
	})
	assert.Nil(t, err)

	V, R, S := tx.RawSignatureValues()
	assert.NotEqual(t, new(big.Int), V)
	assert.NotEqual(t, new(big.Int), R)
	assert.NotEqual(t, new(big.Int), S)
}

func TestServerRemove(t *testing.T) {
	s, cancel := initializeServer()
	defer cancel()

	privateKey, err := crypto.HexToECDSA(strings.Repeat("1", 64))
	assert.Nil(t, err)

	err = s.Generate(ctx, core.GenerateRequest{
		Key:        "key",
		PrivateKey: privateKey,
	})
	assert.Nil(t, err)

	err = s.Remove(ctx, core.RemoveRequest{Key: "key"})
	assert.Nil(t, err)
}
