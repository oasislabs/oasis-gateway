package service

import (
	"context"
	stderr "errors"
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
	if args.Get(1) != nil {
		return 0, args.Get(1).(errors.Err)
	}

	return uint64(args.Int(0)), nil
}

func (c *MockClient) ExecuteServiceAsync(
	ctx context.Context,
	req backend.ExecuteServiceRequest,
) (uint64, errors.Err) {
	args := c.Mock.Called(ctx, req)
	if args.Get(1) != nil {
		return 0, args.Get(1).(errors.Err)
	}

	return uint64(args.Int(0)), nil
}

func (c *MockClient) PollService(
	ctx context.Context,
	req backend.PollServiceRequest,
) (backend.Events, errors.Err) {
	args := c.Mock.Called(ctx, req)
	if args.Get(1) != nil {
		return backend.Events{}, args.Get(1).(errors.Err)
	}

	return args.Get(0).(backend.Events), nil
}

func (c *MockClient) GetPublicKeyService(
	ctx context.Context,
	req backend.GetPublicKeyServiceRequest,
) (backend.GetPublicKeyServiceResponse, errors.Err) {
	args := c.Mock.Called(ctx, req)
	if args.Get(1) != nil {
		return backend.GetPublicKeyServiceResponse{}, args.Get(1).(errors.Err)
	}

	return args.Get(0).(backend.GetPublicKeyServiceResponse), nil
}

func createServiceHandler() ServiceHandler {
	return NewServiceHandler(Services{
		Logger:   Logger,
		Client:   &MockClient{},
		Verifier: auth.TrustedPayloadVerifier{},
	})
}

func TestDeployServiceEmptyData(t *testing.T) {
	ctx := context.WithValue(Context, auth.ContextAuthDataKey, auth.AuthData{
		ExpectedAAD: "",
		SessionKey:  "sessionKey",
	})
	handler := createServiceHandler()

	handler.client.(*MockClient).On("DeployServiceAsync",
		mock.Anything, mock.Anything).Return(0, nil)

	_, err := handler.DeployService(ctx, &DeployServiceRequest{Data: ""})

	assert.Error(t, err)
	baserr := err.(errors.Err)

	assert.Equal(t, "Payload data is too short", baserr.Cause().Error())
	assert.Equal(t, "[7002] error code AuthenticationError with "+
		"desc Failed to verify AAD in transaction data. with cause"+
		" Payload data is too short", baserr.Error())
	assert.Equal(t, errors.ErrFailedAADVerification, baserr.ErrorCode())
}

func TestDeployServiceErr(t *testing.T) {
	ctx := context.WithValue(Context, auth.ContextAuthDataKey, auth.AuthData{
		SessionKey: "sessionKey",
	})
	handler := createServiceHandler()

	handler.client.(*MockClient).On("DeployServiceAsync",
		mock.Anything,
		backend.DeployServiceRequest{
			Data:       "0x00",
			SessionKey: "sessionKey",
		}).Return(0, errors.New(errors.ErrInternalError, stderr.New("made up error")))

	_, err := handler.DeployService(ctx, &DeployServiceRequest{Data: "0x00"})
	assert.Error(t, err)
	baserr := err.(errors.Err)

	assert.Equal(t, "made up error", baserr.Cause().Error())
	assert.Equal(t, "[1000] error code InternalError with desc Internal Error. "+
		"Please check the status of the service. with cause made up error", baserr.Error())
	assert.Equal(t, errors.ErrInternalError, baserr.ErrorCode())
}

func TestDeployServiceOK(t *testing.T) {
	ctx := context.WithValue(Context, auth.ContextAuthDataKey, auth.AuthData{
		SessionKey: "sessionKey",
	})
	handler := createServiceHandler()

	handler.client.(*MockClient).On("DeployServiceAsync",
		mock.Anything,
		backend.DeployServiceRequest{
			Data:       "0x00",
			SessionKey: "sessionKey",
		}).Return(0, nil)

	res, err := handler.DeployService(ctx, &DeployServiceRequest{Data: "0x00"})
	assert.Nil(t, err)
	assert.Equal(t, uint64(0), res.(AsyncResponse).ID)
}
