package core

import (
	"testing"

	mqueue "github.com/oasislabs/oasis-gateway/mqueue/core"
	"github.com/oasislabs/oasis-gateway/rpc"
	"github.com/stretchr/testify/assert"
)

func TestDeserializeDataElement(t *testing.T) {
	p, err := deserializeElement(mqueue.Element{
		Offset: 0,
		Value:  "{\"ID\":0,\"Cause\":{\"errorCode\":1002,\"description\":\"Internal Error. Please check the status of the service.\"}}",
		Type:   ErrorEventType.String(),
	})

	ev := p.(ErrorEvent)
	assert.Nil(t, err)
	assert.Equal(t, uint64(0), ev.ID)
	assert.Equal(t, rpc.Error{
		ErrorCode:   1002,
		Description: "Internal Error. Please check the status of the service.",
	}, ev.Cause)
}
