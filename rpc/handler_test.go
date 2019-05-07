package rpc

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHandlerFunc(t *testing.T) {
	handler := HandlerFunc(func(ctx context.Context, body interface{}) (interface{}, error) {
		return nil, nil
	})

	v, err := handler.Handle(context.Background(), nil)

	assert.Nil(t, err)
	assert.Nil(t, v)
}

func TestEntityFactoryFunc(t *testing.T) {
	factory := EntityFactoryFunc(func() interface{} {
		return nil
	})

	v := factory.Create()

	assert.Nil(t, v)
}
