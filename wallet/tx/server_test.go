package tx

import (
	"context"
	"io/ioutil"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/oasislabs/developer-gateway/log"
	"github.com/oasislabs/developer-gateway/wallet/core"
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
	ctx, cancel := context.WithCancel(context.Background())
	s := NewServer(ctx, logger)

	return s, cancel
}

func TestServerSign(t *testing.T) {
	s, cancel := initializeServer()
	defer cancel()

	tx, err := s.Sign(ctx, core.SignRequest{
		Key:         "key",
		Transaction: tx,
	})
	assert.Nil(t, err)
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
