package service

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExecuteServiceEventEventID(t *testing.T) {
	assert.Equal(t, uint64(1), ExecuteServiceEvent{ID: 1}.EventID())
}

func TestDeployServiceEventEventID(t *testing.T) {
	assert.Equal(t, uint64(1), DeployServiceEvent{ID: 1}.EventID())
}

func TestErrorEventEventID(t *testing.T) {
	assert.Equal(t, uint64(1), ErrorEvent{ID: 1}.EventID())
}

func TestPollServiceRequestType(t *testing.T) {
	assert.Equal(t, Poll, PollServiceRequest{}.Type())
}

func TestDeployServiceRequestType(t *testing.T) {
	assert.Equal(t, Deploy, DeployServiceRequest{}.Type())
}

func TestExecuteServiceRequestType(t *testing.T) {
	assert.Equal(t, Execute, ExecuteServiceRequest{}.Type())
}
