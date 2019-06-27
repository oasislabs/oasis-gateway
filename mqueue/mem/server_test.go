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

func TestServerInsert(t *testing.T) {
	s := NewServer(context.TODO(), Services{Logger: logger})

	offset, err := s.Next(ctx, core.NextRequest{Key: "key"})
	assert.Nil(t, err)

	err = s.Insert(ctx, core.InsertRequest{Key: "key", Element: core.Element{
		Offset: offset,
		Value:  "value",
	}})
	assert.Nil(t, err)
}

func TestServerRetrieve(t *testing.T) {
	s := NewServer(context.TODO(), Services{Logger: logger})

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
			core.Element{
				Offset: uint64(0),
				Value:  "value",
			},
		},
	}, els)
}

func TestServerDiscardKeepPreviousFalse(t *testing.T) {
	s := NewServer(context.TODO(), Services{Logger: logger})

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
			core.Element{
				Offset: uint64(1),
				Value:  "value",
			},
			core.Element{
				Offset: uint64(2),
				Value:  "value",
			},
		},
	}, els)
}

func TestServerDiscardKeepPreviousTrue(t *testing.T) {
	s := NewServer(context.TODO(), Services{Logger: logger})

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

	err = s.Discard(ctx, core.DiscardRequest{
		Key:          "key",
		Offset:       uint64(1),
		Count:        1,
		KeepPrevious: true,
	})
	assert.Nil(t, err)

	var els core.Elements
	els, err = s.Retrieve(ctx, core.RetrieveRequest{Key: "key", Offset: uint64(0), Count: uint(3)})
	assert.Nil(t, err)
	assert.Equal(t, core.Elements{
		Offset: uint64(0),
		Elements: []core.Element{
			core.Element{
				Offset: uint64(0),
				Value:  "value",
			},
			core.Element{
				Offset: uint64(2),
				Value:  "value",
			},
		},
	}, els)
}

func TestServerNext(t *testing.T) {
	s := NewServer(context.TODO(), Services{Logger: logger})

	offset, err := s.Next(ctx, core.NextRequest{Key: "key"})
	assert.Nil(t, err)
	assert.Equal(t, uint64(0), offset)

	offset, err = s.Next(ctx, core.NextRequest{Key: "key"})
	assert.Nil(t, err)
	assert.Equal(t, uint64(1), offset)
}

func TestServerRemove(t *testing.T) {
	s := NewServer(context.TODO(), Services{Logger: logger})

	_, err := s.Next(ctx, core.NextRequest{Key: "key"})
	assert.Nil(t, err)

	err = s.Remove(ctx, core.RemoveRequest{Key: "key"})
	assert.Nil(t, err)
}

func TestServerNextErrLimitReached(t *testing.T) {
	s := NewServer(context.TODO(), Services{Logger: logger})

	var (
		err error
		it  int
	)

	for it = 0; it < 1026 && err == nil; it++ {
		_, err = s.Next(ctx, core.NextRequest{Key: "invalid"})
	}

	assert.Equal(t, "[3001] error code ResourceLimitReached with desc The number of unconfirmed requests has reached its limit. No further requests can be processed until requests are confirmed. with cause window is full and cannot increase its size", err.Error())
	assert.Equal(t, 1024, it)
}

func TestServerName(t *testing.T) {
	s := NewServer(context.TODO(), Services{Logger: logger})
	assert.Equal(t, "mqueue.mem.Server", s.Name())
}

func TestServerStats(t *testing.T) {
	s := NewServer(context.TODO(), Services{Logger: logger})

	assert.Nil(t, s.Stats())
}
