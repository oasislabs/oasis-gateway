package version

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetVersion(t *testing.T) {
	h := NewHandler()

	res, err := h.GetVersion(context.TODO(), &GetVersionRequest{})

	assert.Nil(t, err)
	assert.Equal(t, &GetVersionResponse{
		Version: 0,
	}, res)
}
