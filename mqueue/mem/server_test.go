package mem

import (
	"context"
	"testing"

	"github.com/oasislabs/developer-gateway/log"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestServerRetrieve(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	logger := log.NewLogrus(log.LogrusLoggerProperties{
		Level: logrus.DebugLevel,
	})
	s := NewServer(ctx, logger)

	els, err := s.Retrieve("key", uint64(1), uint(1))
	assert.Equal(t, uint64(1), els.Offset)
	assert.Nil(t, err)
}
