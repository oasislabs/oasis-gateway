package mem

import (
	"context"
	"testing"

	"github.com/oasislabs/developer-gateway/log"
	"github.com/oasislabs/developer-gateway/mqueue/core"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func initializeServer() (*Server, context.CancelFunc) {
	ctx, cancel := context.WithCancel(context.Background())

	logger := log.NewLogrus(log.LogrusLoggerProperties{
		Level: logrus.DebugLevel,
	})
	s := NewServer(ctx, logger)

	return s, cancel
}

func TestServerInsert(t *testing.T) {
	s, cancel := initializeServer()
	defer cancel()

	err := s.Insert("key", core.Element{
		Offset: uint64(1),
		Value: "value",
	})
	assert.Nil(t, err)
}

func TestServerRetrieve(t *testing.T) {
	s, cancel := initializeServer()
	defer cancel()

	els, err := s.Retrieve("key", uint64(1), uint(1))
	assert.Equal(t, els, core.Elements{
		Offset: uint64(1),
		Elements: nil,
	})
	assert.Nil(t, err)
}

func TestServerDiscard(t *testing.T) {
	s, cancel := initializeServer()
	defer cancel()

	err := s.Discard("key", uint64(1))
	assert.Nil(t, err)
}

func TestServerNext(t *testing.T) {
	s, cancel := initializeServer()
	defer cancel()

	offset, err := s.Next("key")
	assert.Equal(t, uint64(0), offset)
	assert.Nil(t, err)
}

func TestServerRemove(t *testing.T) {
	s, cancel := initializeServer()
	defer cancel()

	err := s.Remove("key")
	assert.Nil(t, err)
}