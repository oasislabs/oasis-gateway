package exec

import (
	"context"
	"crypto/ecdsa"
	"io/ioutil"
	"strconv"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/crypto"
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

func TestServerRemove(t *testing.T) {
	s, cancel := initializeServer()
	defer cancel()

	pk, _ := crypto.HexToECDSA(strings.Repeat("1", 64))

	err := s.Remove(ctx, core.RemoveRequest{Key: crypto.PubkeyToAddress(pk.PublicKey).Hex()})
	assert.Nil(t, err)
}
