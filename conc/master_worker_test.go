package conc

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
)

type MockMasterHandler struct {
	created   int32
	destroyed int32
}

func (m *MockMasterHandler) Created() int {
	return int(atomic.LoadInt32(&m.created))
}

func (m *MockMasterHandler) Destroyed() int {
	return int(atomic.LoadInt32(&m.destroyed))
}

func (m *MockMasterHandler) Handle(ctx context.Context, req MasterEvent) error {
	switch req := req.(type) {
	case CreateWorkerEvent:
		req.Props.ErrC = nil
		req.Props.UserData = nil
		req.Props.WorkerHandler = &MockWorkerHandler{}
		atomic.AddInt32(&m.created, 1)
	case DestroyWorkerEvent:
		atomic.AddInt32(&m.destroyed, 1)
	default:
		panic("received unknown master event")
	}

	return nil
}

type MockWorkerHandler struct {
	Value int
}

func (m *MockWorkerHandler) Handle(ctx context.Context, req WorkerEvent) (interface{}, error) {
	switch req := req.(type) {
	case RequestWorkerEvent:
		m.Value = req.Value.(int) + 1
		return m.Value, nil
	case ErrorWorkerEvent:
		return nil, req.Error
	default:
		panic("received unknown worker event")
	}
}

func TestNewMaster(t *testing.T) {
	master := NewMaster(MasterProps{
		MasterHandler: &MockMasterHandler{},
	})

	err := master.Start(context.Background())
	assert.Nil(t, err)

	err = master.Stop()
	assert.Nil(t, err)
}

func TestNewMasterStartTwice(t *testing.T) {
	master := NewMaster(MasterProps{
		MasterHandler: &MockMasterHandler{},
	})

	err := master.Start(context.Background())
	assert.Nil(t, err)
	defer func() {
		err := master.Stop()
		assert.Nil(t, err)
	}()

	err = master.Start(context.Background())
	assert.Error(t, err)
}

func TestNewMasterStopTwice(t *testing.T) {
	master := NewMaster(MasterProps{
		MasterHandler: &MockMasterHandler{},
	})

	err := master.Start(context.Background())
	assert.Nil(t, err)

	err = master.Stop()
	assert.Nil(t, err)

	err = master.Stop()
	assert.Error(t, err)
}

func TestNewMasterStopWithoutStart(t *testing.T) {
	master := NewMaster(MasterProps{
		MasterHandler: &MockMasterHandler{},
	})

	err := master.Stop()
	assert.Error(t, err)
}

func TestMasterWorkerExists(t *testing.T) {
	ctx := context.Background()
	handler := &MockMasterHandler{}
	master := NewMaster(MasterProps{
		MasterHandler: handler,
	})

	err := master.Start(ctx)
	assert.Nil(t, err)

	ok := master.Exists(ctx, "1")
	assert.False(t, ok)

	err = master.Create(ctx, "1", nil)
	assert.Nil(t, err)

	ok = master.Exists(ctx, "1")
	assert.True(t, ok)

	err = master.Destroy(ctx, "1")
	assert.Nil(t, err)

	ok = master.Exists(ctx, "1")
	assert.False(t, ok)

	err = master.Stop()
	assert.Nil(t, err)

	assert.Equal(t, 1, handler.Created())
	assert.Equal(t, 1, handler.Destroyed())
}

func TestMasterWorkerRequest(t *testing.T) {
	ctx := context.Background()
	handler := &MockMasterHandler{}
	master := NewMaster(MasterProps{
		MasterHandler: handler,
	})

	err := master.Start(ctx)
	assert.Nil(t, err)

	err = master.Create(ctx, "1", nil)
	assert.Nil(t, err)

	v, err := master.Request(ctx, "1", 0)
	assert.Nil(t, err)
	assert.Equal(t, 1, v)

	err = master.Destroy(ctx, "1")
	assert.Nil(t, err)

	err = master.Stop()
	assert.Nil(t, err)

	assert.Equal(t, 1, handler.Created())
	assert.Equal(t, 1, handler.Destroyed())
}

func TestMasterStopShutdownWorkers(t *testing.T) {
	ctx := context.Background()
	handler := &MockMasterHandler{}
	master := NewMaster(MasterProps{
		MasterHandler: handler,
	})

	err := master.Start(ctx)
	assert.Nil(t, err)

	err = master.Create(ctx, "1", nil)
	assert.Nil(t, err)

	err = master.Stop()
	assert.Nil(t, err)

	assert.Equal(t, 1, handler.Created())
	assert.Equal(t, 1, handler.Destroyed())
}

func TestMasterWorkerPanicOnCreate(t *testing.T) {
	ctx := context.Background()
	handler := MasterHandlerFunc(func(ctx context.Context, ev MasterEvent) error {
		if ev, ok := ev.(CreateWorkerEvent); ok {
			ev.Props.WorkerHandler = WorkerHandlerFunc(func(ctx context.Context, ev WorkerEvent) (interface{}, error) {
				panic("error")
			})
		}

		return nil
	})
	master := NewMaster(MasterProps{
		MasterHandler: handler,
	})

	err := master.Start(ctx)
	assert.Nil(t, err)

	err = master.Create(ctx, "1", nil)
	assert.Nil(t, err)

	err = master.Stop()
	assert.Nil(t, err)
}

func TestMasterHandlerErrorOnCreate(t *testing.T) {
	ctx := context.Background()
	handler := MasterHandlerFunc(func(ctx context.Context, ev MasterEvent) error {
		return errors.New("error")
	})
	master := NewMaster(MasterProps{
		MasterHandler: handler,
	})

	err := master.Start(ctx)
	assert.Nil(t, err)

	err = master.Create(ctx, "1", nil)
	assert.Error(t, err)
}

func TestMasterHandlerErrorOnDestroy(t *testing.T) {
	ctx := context.Background()
	handler := MasterHandlerFunc(func(ctx context.Context, ev MasterEvent) error {
		switch req := ev.(type) {
		case CreateWorkerEvent:
			req.Props.ErrC = nil
			req.Props.UserData = nil
			req.Props.WorkerHandler = &MockWorkerHandler{}
		case DestroyWorkerEvent:
			return errors.New("error")
		default:
			panic("received unknown master event")
		}

		return nil
	})
	master := NewMaster(MasterProps{
		MasterHandler: handler,
	})

	err := master.Start(ctx)
	assert.Nil(t, err)

	err = master.Create(ctx, "1", nil)
	assert.Nil(t, err)

	err = master.Destroy(ctx, "1")
	assert.Error(t, err)

	err = master.Stop()
	assert.Nil(t, err)
}

func TestMasterHandlerPanicOnDestroy(t *testing.T) {
	ctx := context.Background()
	handler := MasterHandlerFunc(func(ctx context.Context, ev MasterEvent) error {
		switch req := ev.(type) {
		case CreateWorkerEvent:
			req.Props.ErrC = nil
			req.Props.UserData = nil
			req.Props.WorkerHandler = &MockWorkerHandler{}
		case DestroyWorkerEvent:
			panic("error")
		default:
			panic("received unknown master event")
		}

		return nil
	})
	master := NewMaster(MasterProps{
		MasterHandler: handler,
	})

	err := master.Start(ctx)
	assert.Nil(t, err)

	err = master.Create(ctx, "1", nil)
	assert.Nil(t, err)

	err = master.Destroy(ctx, "1")
	assert.Error(t, err)

	err = master.Stop()
	assert.Nil(t, err)
}
