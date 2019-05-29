package mem

import (
	"context"
	"io/ioutil"
	"testing"

	"github.com/oasislabs/developer-gateway/log"
	"github.com/oasislabs/developer-gateway/mqueue/core"
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

func TestServerInsert(t *testing.T) {
	s, cancel := initializeServer()
	defer cancel()

	offset, err := s.Next(ctx, core.NextRequest{Key: "key"})
	assert.Nil(t, err)

	err = s.Insert(ctx, core.InsertRequest{Key: "key", Element: core.Element{
		Offset: offset,
		Value:  "value",
	}})
	assert.Nil(t, err)
}

func TestServerRetrieve(t *testing.T) {
	s, cancel := initializeServer()
	defer cancel()

	els, err := s.Retrieve(ctx, core.RetrieveRequest{Key: "key", Offset: uint64(1), Count: uint(1)})
	assert.Nil(t, err)
	assert.Equal(t, core.Elements{
		Offset:   uint64(0),
		Elements: []core.Element{},
	}, els)

	var offset uint64
	offset, err = s.Next(ctx, core.NextRequest{Key: "key"})
	assert.Nil(t, err)

	err = s.Insert(ctx, core.InsertRequest{Key: "key", Element: core.Element{
		Offset: offset,
		Value:  "value",
	}})
	assert.Nil(t, err)

	els, err = s.Retrieve(ctx, core.RetrieveRequest{Key: "key", Offset: offset, Count: uint(1)})
	assert.Nil(t, err)
	assert.Equal(t, core.Elements{
		Offset: offset,
		Elements: []core.Element{
			{
				Offset: uint64(0),
				Value:  "value",
			},
		},
	}, els)
}

func TestServerDiscard(t *testing.T) {
	s, cancel := initializeServer()
	defer cancel()

	var offset uint64
	var err error
	for i := 0; i < 3; i++ {
		offset, err = s.Next(ctx, core.NextRequest{Key: "key"})
		assert.Nil(t, err)

		err = s.Insert(ctx, core.InsertRequest{Key: "key", Element: core.Element{
			Offset: offset,
			Value:  "value",
		}})
		assert.Nil(t, err)
	}

	err = s.Discard(ctx, core.DiscardRequest{Key: "key", Offset: uint64(1)})
	assert.Nil(t, err)

	var els core.Elements
	els, err = s.Retrieve(ctx, core.RetrieveRequest{Key: "key", Offset: uint64(0), Count: uint(2)})
	assert.Nil(t, err)
	assert.Equal(t, core.Elements{
		Offset: uint64(1),
		Elements: []core.Element{
			{
				Offset: uint64(1),
				Value:  "value",
			},
			{
				Offset: uint64(2),
				Value:  "value",
			},
		},
	}, els)
}

func TestServerNext(t *testing.T) {
	s, cancel := initializeServer()
	defer cancel()

	offset, err := s.Next(ctx, core.NextRequest{Key: "key"})
	assert.Nil(t, err)
	assert.Equal(t, uint64(0), offset)

	offset, err = s.Next(ctx, core.NextRequest{Key: "key"})
	assert.Nil(t, err)
	assert.Equal(t, uint64(1), offset)
}

func TestServerRemove(t *testing.T) {
	s, cancel := initializeServer()
	defer cancel()

	_, err := s.Next(ctx, core.NextRequest{Key: "key"})
	assert.Nil(t, err)

	err = s.Remove(ctx, core.RemoveRequest{Key: "key"})
	assert.Nil(t, err)
}
