package mem

import (
	"context"
	"errors"
	"testing"

	"github.com/oasislabs/oasis-gateway/concurrent"
	"github.com/stretchr/testify/assert"
)

type InvalidEvent struct{}

func (e InvalidEvent) GetWorker() *concurrent.Worker {
	return nil
}

func TestMessageHandlerHandleError(t *testing.T) {
	handler := NewMessageHandler("key")

	v, err := handler.handle(context.TODO(), concurrent.ErrorWorkerEvent{
		Worker: nil,
		Error:  errors.New("error"),
	})

	assert.Nil(t, v)
	assert.Error(t, err)
}

func TestMessageHandlerHandleUnknown(t *testing.T) {
	handler := NewMessageHandler("key")

	assert.Panics(t, func() {
		_, _ = handler.handle(context.TODO(), InvalidEvent{})
	})
}

func TestMessageHandlerHandleWorkerRequestUnknown(t *testing.T) {
	handler := NewMessageHandler("key")

	assert.Panics(t, func() {
		_, _ = handler.handle(context.TODO(), concurrent.RequestWorkerEvent{
			Worker: nil,
			Value:  "unknown",
		})
	})
}
