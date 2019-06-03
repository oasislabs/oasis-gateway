package service

import (
	"context"
	"io/ioutil"
	"testing"

	auth "github.com/oasislabs/developer-gateway/auth/core"
	backend "github.com/oasislabs/developer-gateway/backend/core"
	"github.com/oasislabs/developer-gateway/errors"
	"github.com/oasislabs/developer-gateway/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

var Context = context.Background()

var Logger = log.NewLogrus(log.LogrusLoggerProperties{
	Output: ioutil.Discard,
})

type MockClient struct {
	mock.Mock
}

func (c *MockClient) DeployServiceAsync(
	ctx context.Context,
	req backend.DeployServiceRequest,
) (uint64, errors.Err) {
	args := c.Mock.Called(ctx, req)
	return uint64(args.Int(0)), args.Get(1).(errors.Err)
}

func (c *MockClient) ExecuteServiceAsync(
	ctx context.Context,
	req backend.ExecuteServiceRequest,
) (uint64, errors.Err) {
	args := c.Mock.Called(ctx, req)
	return uint64(args.Int(0)), args.Get(1).(errors.Err)
}

func (c *MockClient) PollService(
	ctx context.Context,
	req backend.PollServiceRequest,
) (backend.Events, errors.Err) {
	args := c.Mock.Called(ctx, req)
	return args.Get(0).(backend.Events), args.Get(1).(errors.Err)
}

func (c *MockClient) GetPublicKeyService(
	ctx context.Context,
	req backend.GetPublicKeyServiceRequest,
) (backend.GetPublicKeyServiceResponse, errors.Err) {
	args := c.Mock.Called(ctx, req)
	return args.Get(0).(backend.GetPublicKeyServiceResponse), args.Get(1).(errors.Err)
}

func createServiceHandler() ServiceHandler {
	return NewServiceHandler(Services{
		Logger:   Logger,
		Client:   &MockClient{},
		Verifier: auth.TrustedPayloadVerifier{},
	})
}

func TestDeployServiceEmptyData(t *testing.T) {
	ctx := context.WithValue(Context, auth.ContextAuthDataKey, auth.AuthData{})
	handler := createServiceHandler()

	_, err := handler.DeployService(ctx, &DeployServiceRequest{Data: ""})

	assert.Error(t, err)
	baserr := err.(errors.Err)

	assert.Equal(t, "Payload data is too short", baserr.Cause().Error())
	assert.Equal(t, "[7002] error code AuthenticationError with desc Failed to verify AAD in transaction data. with cause Payload data is too short", baserr.Error())
	assert.Equal(t, errors.ErrFailedAADVerification, baserr.ErrorCode())
}
