package service

import (
	"context"
	stderr "errors"
	"io/ioutil"
	"testing"

	auth "github.com/oasislabs/developer-gateway/auth/core"
	insecureauth "github.com/oasislabs/developer-gateway/auth/insecure"
	backend "github.com/oasislabs/developer-gateway/backend/core"
	"github.com/oasislabs/developer-gateway/errors"
	"github.com/oasislabs/developer-gateway/log"
	"github.com/oasislabs/developer-gateway/rpc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

var Context = context.TODO()

var Logger = log.NewLogrus(log.LogrusLoggerProperties{
	Output: ioutil.Discard,
})

type InvalidEvent struct{}

func (e InvalidEvent) EventID() uint64 {
	return 0
}

func (e InvalidEvent) EventType() backend.EventType {
	return backend.DataEventType
}

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

func (c *MockClient) GetCode(
	ctx context.Context,
	req backend.GetCodeRequest,
) (backend.GetCodeResponse, errors.Err) {
	args := c.Mock.Called(ctx, req)
	if args.Get(1) != nil {
		return backend.GetCodeResponse{}, args.Get(1).(errors.Err)
	}

	return args.Get(0).(backend.GetCodeResponse), nil
}

func (c *MockClient) GetPublicKey(
	ctx context.Context,
	req backend.GetPublicKeyRequest,
) (backend.GetPublicKeyResponse, errors.Err) {
	args := c.Mock.Called(ctx, req)
	if args.Get(1) != nil {
		return backend.GetPublicKeyResponse{}, args.Get(1).(errors.Err)
	}

	return args.Get(0).(backend.GetPublicKeyResponse), nil
}

func createServiceHandler() ServiceHandler {
	return NewServiceHandler(Services{
		Logger:   Logger,
		Client:   &MockClient{},
		Verifier: insecureauth.InsecureAuth{},
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

func TestExecuteServiceEmptyData(t *testing.T) {
	ctx := context.WithValue(Context, auth.ContextAuthDataKey, auth.AuthData{
		ExpectedAAD: "",
		SessionKey:  "sessionKey",
	})
	handler := createServiceHandler()

	handler.client.(*MockClient).On("ExecuteServiceAsync",
		mock.Anything, mock.Anything).Return(0, nil)

	_, err := handler.ExecuteService(ctx, &ExecuteServiceRequest{
		Data:    "",
		Address: "0x00",
	})

	assert.Error(t, err)
	baserr := err.(errors.Err)

	assert.Equal(t, "Payload data is too short", baserr.Cause().Error())
	assert.Equal(t, errors.ErrFailedAADVerification, baserr.ErrorCode())
}

func TestExecuteServiceEmptyAddress(t *testing.T) {
	ctx := context.WithValue(Context, auth.ContextAuthDataKey, auth.AuthData{
		ExpectedAAD: "",
		SessionKey:  "sessionKey",
	})
	handler := createServiceHandler()

	handler.client.(*MockClient).On("ExecuteServiceAsync",
		mock.Anything, mock.Anything).Return(0, nil)

	_, err := handler.ExecuteService(ctx, &ExecuteServiceRequest{
		Data:    "0x00",
		Address: "",
	})

	assert.Error(t, err)
	baserr := err.(errors.Err)

	assert.Nil(t, baserr.Cause())
	assert.Equal(t, "[2006] error code InputError with desc Provided invalid address.", baserr.Error())
	assert.Equal(t, errors.ErrInvalidAddress, baserr.ErrorCode())
}

func TestExecuteServiceErr(t *testing.T) {
	ctx := context.WithValue(Context, auth.ContextAuthDataKey, auth.AuthData{
		SessionKey: "sessionKey",
	})
	handler := createServiceHandler()

	handler.client.(*MockClient).On("ExecuteServiceAsync",
		mock.Anything,
		backend.ExecuteServiceRequest{
			Data:       "0x00",
			Address:    "0x00",
			SessionKey: "sessionKey",
		}).Return(0, errors.New(errors.ErrInternalError, stderr.New("made up error")))

	_, err := handler.ExecuteService(ctx, &ExecuteServiceRequest{
		Data:    "0x00",
		Address: "0x00",
	})
	assert.Error(t, err)
	baserr := err.(errors.Err)

	assert.Equal(t, "made up error", baserr.Cause().Error())
	assert.Equal(t, errors.ErrInternalError, baserr.ErrorCode())
}

func TestExecuteServiceOK(t *testing.T) {
	ctx := context.WithValue(Context, auth.ContextAuthDataKey, auth.AuthData{
		SessionKey: "sessionKey",
	})
	handler := createServiceHandler()

	handler.client.(*MockClient).On("ExecuteServiceAsync",
		mock.Anything,
		backend.ExecuteServiceRequest{
			Data:       "0x00",
			Address:    "0x00",
			SessionKey: "sessionKey",
		}).Return(0, nil)

	res, err := handler.ExecuteService(ctx, &ExecuteServiceRequest{
		Data:    "0x00",
		Address: "0x00",
	})
	assert.Nil(t, err)
	assert.Equal(t, uint64(0), res.(AsyncResponse).ID)
}

func TestPollServiceErr(t *testing.T) {
	ctx := context.WithValue(Context, auth.ContextAuthDataKey, auth.AuthData{
		SessionKey: "sessionKey",
	})
	handler := createServiceHandler()

	handler.client.(*MockClient).On("PollService",
		mock.Anything,
		backend.PollServiceRequest{
			Offset:          0,
			Count:           10,
			DiscardPrevious: false,
			SessionKey:      "sessionKey",
		}).Return(nil, errors.New(errors.ErrInternalError, stderr.New("made up error")))

	_, err := handler.PollService(ctx, &PollServiceRequest{
		Offset:          0,
		Count:           10,
		DiscardPrevious: false,
	})

	assert.Error(t, err)
	baserr := err.(errors.Err)

	assert.Equal(t, "made up error", baserr.Cause().Error())
	assert.Equal(t, errors.ErrInternalError, baserr.ErrorCode())
}

func TestPollServiceDeployOK(t *testing.T) {
	ctx := context.WithValue(Context, auth.ContextAuthDataKey, auth.AuthData{
		SessionKey: "sessionKey",
	})
	handler := createServiceHandler()

	handler.client.(*MockClient).On("PollService",
		mock.Anything,
		backend.PollServiceRequest{
			Offset:          0,
			Count:           10,
			DiscardPrevious: false,
			SessionKey:      "sessionKey",
		}).Return(backend.Events{
		Offset: 0,
		Events: []backend.Event{backend.DeployServiceResponse{ID: 0, Address: "0x00"}}}, nil)

	res, err := handler.PollService(ctx, &PollServiceRequest{
		Offset:          0,
		Count:           10,
		DiscardPrevious: false,
	})
	assert.Nil(t, err)

	evs := res.(PollServiceResponse)
	assert.Equal(t, 1, len(evs.Events))
	assert.Equal(t, uint64(0), evs.Offset)
	assert.Equal(t, DeployServiceEvent{
		ID:      0,
		Address: "0x00",
	}, evs.Events[0])
}

func TestPollServiceExecuteOK(t *testing.T) {
	ctx := context.WithValue(Context, auth.ContextAuthDataKey, auth.AuthData{
		SessionKey: "sessionKey",
	})
	handler := createServiceHandler()

	handler.client.(*MockClient).On("PollService",
		mock.Anything,
		backend.PollServiceRequest{
			Offset:          0,
			Count:           10,
			DiscardPrevious: false,
			SessionKey:      "sessionKey",
		}).Return(backend.Events{
		Offset: 0,
		Events: []backend.Event{backend.ExecuteServiceResponse{ID: 0, Address: "0x00", Output: "0x00"}}}, nil)

	res, err := handler.PollService(ctx, &PollServiceRequest{
		Offset:          0,
		Count:           0,
		DiscardPrevious: false,
	})
	assert.Nil(t, err)

	evs := res.(PollServiceResponse)
	assert.Equal(t, 1, len(evs.Events))
	assert.Equal(t, uint64(0), evs.Offset)
	assert.Equal(t, ExecuteServiceEvent{
		ID:      0,
		Address: "0x00",
		Output:  "0x00",
	}, evs.Events[0])
}

func TestPollServiceErrorOK(t *testing.T) {
	ctx := context.WithValue(Context, auth.ContextAuthDataKey, auth.AuthData{
		SessionKey: "sessionKey",
	})
	handler := createServiceHandler()

	handler.client.(*MockClient).On("PollService",
		mock.Anything,
		backend.PollServiceRequest{
			Offset:          0,
			Count:           10,
			DiscardPrevious: false,
			SessionKey:      "sessionKey",
		}).Return(backend.Events{
		Offset: 0,
		Events: []backend.Event{backend.ErrorEvent{ID: 0, Cause: rpc.Error{}}}}, nil)

	res, err := handler.PollService(ctx, &PollServiceRequest{
		Offset:          0,
		Count:           10,
		DiscardPrevious: false,
	})
	assert.Nil(t, err)

	evs := res.(PollServiceResponse)
	assert.Equal(t, 1, len(evs.Events))
	assert.Equal(t, uint64(0), evs.Offset)
	assert.Equal(t, ErrorEvent{
		ID:    0,
		Cause: rpc.Error{},
	}, evs.Events[0])
}

func TestGetCodeEmptyAddress(t *testing.T) {
	ctx := context.WithValue(Context, auth.ContextAuthDataKey, auth.AuthData{
		SessionKey: "sessionKey",
	})
	handler := createServiceHandler()

	handler.client.(*MockClient).On("GetCode",
		mock.Anything,
		backend.GetCodeRequest{
			Address: "0x00",
		}).Return(nil, nil)

	_, err := handler.GetCode(ctx, &GetCodeRequest{
		Address: "",
	})

	assert.Error(t, err)
	baserr := err.(errors.Err)

	assert.Equal(t, "address field has not been set", baserr.Cause().Error())
	assert.Equal(t, errors.ErrInvalidAddress, baserr.ErrorCode())
}

func TestGetCodeEmptyErr(t *testing.T) {
	ctx := context.WithValue(Context, auth.ContextAuthDataKey, auth.AuthData{
		SessionKey: "sessionKey",
	})
	handler := createServiceHandler()

	handler.client.(*MockClient).On("GetCode",
		mock.Anything,
		backend.GetCodeRequest{
			Address: "0x00",
		}).Return(nil, errors.New(errors.ErrInternalError, stderr.New("made up error")))

	_, err := handler.GetCode(ctx, &GetCodeRequest{
		Address: "0x00",
	})

	assert.Error(t, err)
	baserr := err.(errors.Err)

	assert.Equal(t, "made up error", baserr.Cause().Error())
	assert.Equal(t, errors.ErrInternalError, baserr.ErrorCode())
}

func TestGetCodeEmptyOK(t *testing.T) {
	ctx := context.WithValue(Context, auth.ContextAuthDataKey, auth.AuthData{
		SessionKey: "sessionKey",
	})
	handler := createServiceHandler()

	handler.client.(*MockClient).On("GetCode",
		mock.Anything,
		backend.GetCodeRequest{
			Address: "0x00",
		}).Return(backend.GetCodeResponse{
		Code:    []byte("service implementation"),
		Address: "0x00",
	}, nil)

	res, err := handler.GetCode(ctx, &GetCodeRequest{
		Address: "0x00",
	})
	assert.Nil(t, err)
	assert.Equal(t, GetCodeResponse{
		Code:    []byte("service implementation"),
		Address: "0x00",
	}, res)
}

func TestGetPublicKeyEmptyAddress(t *testing.T) {
	ctx := context.WithValue(Context, auth.ContextAuthDataKey, auth.AuthData{
		SessionKey: "sessionKey",
	})
	handler := createServiceHandler()

	handler.client.(*MockClient).On("GetPublicKey",
		mock.Anything,
		backend.GetPublicKeyRequest{
			Address: "0x00",
		}).Return(nil, nil)

	_, err := handler.GetPublicKey(ctx, &GetPublicKeyRequest{
		Address: "",
	})

	assert.Error(t, err)
	baserr := err.(errors.Err)

	assert.Equal(t, "address field has not been set", baserr.Cause().Error())
	assert.Equal(t, errors.ErrInvalidAddress, baserr.ErrorCode())
}

func TestGetPublicKeyEmptyErr(t *testing.T) {
	ctx := context.WithValue(Context, auth.ContextAuthDataKey, auth.AuthData{
		SessionKey: "sessionKey",
	})
	handler := createServiceHandler()

	handler.client.(*MockClient).On("GetPublicKey",
		mock.Anything,
		backend.GetPublicKeyRequest{
			Address: "0x00",
		}).Return(nil, errors.New(errors.ErrInternalError, stderr.New("made up error")))

	_, err := handler.GetPublicKey(ctx, &GetPublicKeyRequest{
		Address: "0x00",
	})

	assert.Error(t, err)
	baserr := err.(errors.Err)

	assert.Equal(t, "made up error", baserr.Cause().Error())
	assert.Equal(t, errors.ErrInternalError, baserr.ErrorCode())
}

func TestGetPublicKeyEmptyOK(t *testing.T) {
	ctx := context.WithValue(Context, auth.ContextAuthDataKey, auth.AuthData{
		SessionKey: "sessionKey",
	})
	handler := createServiceHandler()

	handler.client.(*MockClient).On("GetPublicKey",
		mock.Anything,
		backend.GetPublicKeyRequest{
			Address: "0x00",
		}).Return(backend.GetPublicKeyResponse{
		Timestamp: 1234567890987654321,
		Address:   "0x00",
		PublicKey: "0x01",
		Signature: "0x02",
	}, nil)

	res, err := handler.GetPublicKey(ctx, &GetPublicKeyRequest{
		Address: "0x00",
	})
	assert.Nil(t, err)
	assert.Equal(t, GetPublicKeyResponse{
		Timestamp: 1234567890987654321,
		Address:   "0x00",
		PublicKey: "0x01",
		Signature: "0x02",
	}, res)
}

func TestMapUnkonwnEvent(t *testing.T) {
	handler := createServiceHandler()

	assert.Panics(t, func() {
		handler.mapEvent(InvalidEvent{})
	})
}

func TestNewServiceHandlerNoLogger(t *testing.T) {
	assert.Panics(t, func() {
		NewServiceHandler(Services{
			Client: &MockClient{},
			Logger: nil,
		})
	})
}

func TestNewServiceHandlerNoClient(t *testing.T) {
	assert.Panics(t, func() {
		NewServiceHandler(Services{
			Client: nil,
			Logger: Logger,
		})
	})
}

func TestNewServiceHandlerOK(t *testing.T) {
	h := NewServiceHandler(Services{
		Client: &MockClient{},
		Logger: Logger,
	})

	assert.NotNil(t, h)
}

func TestBindHandlerOK(t *testing.T) {
	binder := rpc.NewHttpBinder(rpc.HttpBinderProperties{
		Encoder: rpc.JsonEncoder{},
		Logger:  Logger,
		HandlerFactory: rpc.HttpHandlerFactoryFunc(func(factory rpc.EntityFactory, handler rpc.Handler) rpc.HttpMiddleware {
			return rpc.NewHttpJsonHandler(rpc.HttpJsonHandlerProperties{
				Limit:   1 << 16,
				Handler: handler,
				Logger:  Logger,
				Factory: factory,
			})
		}),
	})

	BindHandler(Services{
		Client: &MockClient{},
		Logger: Logger,
	}, binder)

	router := binder.Build()

	assert.True(t, router.HasHandler("/v0/api/service/deploy", "POST"))
	assert.True(t, router.HasHandler("/v0/api/service/execute", "POST"))
	assert.True(t, router.HasHandler("/v0/api/service/poll", "POST"))
	assert.True(t, router.HasHandler("/v0/api/service/getPublicKey", "GET"))
}
