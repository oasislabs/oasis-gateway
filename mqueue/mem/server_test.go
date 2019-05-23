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

	offset, err := s.Next("key")
	assert.Nil(t, err)

	err = s.Insert("key", core.Element{
			Offset: offset,
			Value: "value",
	})
	assert.Nil(t, err)
}

func TestServerRetrieve(t *testing.T) {
	s, cancel := initializeServer()
	defer cancel()

	els, err := s.Retrieve("key", uint64(1), uint(1))
	assert.Nil(t, err)
	assert.Equal(t, els, core.Elements{
		Offset: uint64(1),
		Elements: nil,
	})

	var offset uint64
	offset, err = s.Next("key")
	assert.Nil(t, err)

	err = s.Insert("key", core.Element{
		Offset: offset,
		Value: "value",
	})
	assert.Nil(t, err)

	els, err = s.Retrieve("key", offset, uint(1))
	assert.Nil(t, err)
	assert.Equal(t, core.Elements{
		Offset: offset,
		Elements: []core.Element{
			core.Element{
				Offset: offset,
				Value: "value",
			},
		},
	}, els)
}

func TestServerDiscard(t *testing.T) {
	s, cancel := initializeServer()
	defer cancel()

	offset, err := s.Next("key")
	assert.Nil(t, err)

	err = s.Insert("key", core.Element{
		Offset: offset,
		Value: "value0",
	})
	assert.Nil(t, err)

	offset, err = s.Next("key")
	assert.Nil(t, err)

	err = s.Insert("key", core.Element{
		Offset: offset,
		Value: "value1",
	})
	assert.Nil(t, err)

	offset, err = s.Next("key")
	assert.Nil(t, err)

	err = s.Insert("key", core.Element{
		Offset: offset,
		Value: "value2",
	})
	assert.Nil(t, err)

	err = s.Discard("key", uint64(1))
	assert.Nil(t, err)

	var els core.Elements
	els, err = s.Retrieve("key", uint64(0), uint(2))
	assert.Nil(t, err)
	assert.Equal(t, core.Elements{
		Offset: uint64(1),
		Elements: []core.Element{
			core.Element{
				Offset: uint64(1),
				Value: "value1",
			},
			core.Element{
				Offset: uint64(2),
				Value: "value2",
			},
		},
	}, els)
}

func TestServerNext(t *testing.T) {
	s, cancel := initializeServer()
	defer cancel()

	offset, err := s.Next("key")
	assert.Nil(t, err)
	assert.Equal(t, uint64(0), offset)
	
	offset, err = s.Next("key")
	assert.Nil(t, err)
	assert.Equal(t, uint64(1), offset)
}

func TestServerRemove(t *testing.T) {
	s, cancel := initializeServer()
	defer cancel()

	_, err := s.Next("key")
	assert.Nil(t, err)

	err = s.Remove("key")
	assert.Nil(t, err)
}
