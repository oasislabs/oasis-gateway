package tests

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/oasislabs/developer-gateway/api/v0/service"
	"github.com/oasislabs/developer-gateway/eth"
	"github.com/oasislabs/developer-gateway/eth/ethtest"
	"github.com/oasislabs/developer-gateway/rpc"
	"github.com/oasislabs/developer-gateway/tests/apitest"
	"github.com/oasislabs/developer-gateway/tests/gatewaytest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

type ServicesTestSuite struct {
	suite.Suite
	ethclient *ethtest.MockClient
	client    *apitest.ServiceClient
}

func (s *ServicesTestSuite) SetupTest() {
	provider, err := gatewaytest.NewServices(context.TODO(), Config)
	if err != nil {
		panic(err)
	}

	s.ethclient = provider.MustGet(reflect.TypeOf((*eth.Client)(nil)).Elem()).(*ethtest.MockClient)

	router := gatewaytest.NewPublicRouter(provider)
	s.client = apitest.NewServiceClient(router)
}

func (s *ServicesTestSuite) TestDeployServiceEmptyData() {
	ethtest.ImplementMock(s.ethclient)

	_, err := s.client.DeployServiceSync(context.TODO(), service.DeployServiceRequest{
		Data: "",
	})

	assert.Equal(s.T(), &rpc.Error{ErrorCode: 7002, Description: "Failed to verify AAD in transaction data."}, err)
}

func (s *ServicesTestSuite) TestDeployServiceErrEstimateGas() {
	ethtest.ImplementMockWithOverwrite(s.ethclient,
		ethtest.MockMethods{
			"EstimateGas": ethtest.MockMethod{
				Arguments: []interface{}{mock.Anything, mock.Anything},
				Return:    []interface{}{uint64(0), errors.New("error")},
			},
		})

	ev, err := s.client.DeployServiceSync(context.TODO(), service.DeployServiceRequest{
		Data: "0x0000000000000000000000000000000000000000",
	})

	assert.Nil(s.T(), err)
	assert.Equal(s.T(), service.ErrorEvent{
		ID: 0x0,
		Cause: rpc.Error{
			ErrorCode:   1002,
			Description: "Internal Error. Please check the status of the service.",
		}}, ev)
}

func (s *ServicesTestSuite) TestDeployServiceOK() {
	ethtest.ImplementMock(s.ethclient)

	ev, err := s.client.DeployServiceSync(context.TODO(), service.DeployServiceRequest{
		Data: "0x0000000000000000000000000000000000000000",
	})

	assert.Nil(s.T(), err)
	assert.Equal(s.T(), service.DeployServiceEvent{
		ID:      0,
		Address: "0x0000000000000000000000000000000000000000",
	}, ev)
}

func (s *ServicesTestSuite) TestExecuteServiceEmptyAddress() {
	ethtest.ImplementMock(s.ethclient)

	_, err := s.client.ExecuteServiceSync(context.Background(), service.ExecuteServiceRequest{
		Address: "",
		Data:    "0x0000000000000000000000000000000000000000",
	})

	assert.Error(s.T(), err)
	assert.Equal(s.T(), &rpc.Error{ErrorCode: 2006, Description: "Provided invalid address."}, err)
}

func (s *ServicesTestSuite) TestExecuteServiceEmptyTransactionData() {
	ethtest.ImplementMock(s.ethclient)

	_, err := s.client.ExecuteServiceSync(context.Background(), service.ExecuteServiceRequest{
		Address: "0x0000000000000000000000000000000000000000",
		Data:    "",
	})

	assert.Error(s.T(), err)
	assert.Equal(s.T(), &rpc.Error{ErrorCode: 7002, Description: "Failed to verify AAD in transaction data."}, err)
}

func (s *ServicesTestSuite) TestExecuteServiceErrEstimateGas() {
	ethtest.ImplementMockWithOverwrite(s.ethclient,
		ethtest.MockMethods{
			"EstimateGas": ethtest.MockMethod{
				Arguments: []interface{}{mock.Anything, mock.Anything},
				Return:    []interface{}{uint64(0), errors.New("error")},
			},
		})

	ev, err := s.client.ExecuteServiceSync(context.TODO(), service.ExecuteServiceRequest{
		Address: "0x0000000000000000000000000000000000000000",
		Data:    "0x0000000000000000000000000000000000000000",
	})
	assert.Nil(s.T(), err)
	assert.Equal(s.T(), service.ErrorEvent{
		ID: 0x0,
		Cause: rpc.Error{
			ErrorCode:   1002,
			Description: "Internal Error. Please check the status of the service.",
		}}, ev)
}

func (s *ServicesTestSuite) TestExecuteServiceOK() {
	ethtest.ImplementMock(s.ethclient)

	ev, err := s.client.ExecuteServiceSync(context.TODO(), service.ExecuteServiceRequest{
		Address: "0x0000000000000000000000000000000000000000",
		Data:    "0x0000000000000000000000000000000000000000",
	})
	assert.Nil(s.T(), err)
	assert.Equal(s.T(), service.ExecuteServiceEvent{
		ID:      0,
		Address: "0x0000000000000000000000000000000000000000",
		Output:  "0x73756363657373",
	}, ev)
}

func (s *ServicesTestSuite) TestExecuteServiceErrStatus0() {
	ethtest.ImplementMockWithOverwrite(s.ethclient,
		ethtest.MockMethods{
			"SendTransaction": ethtest.MockMethod{
				Arguments: []interface{}{mock.Anything, mock.Anything},
				Return: []interface{}{
					eth.SendTransactionResponse{
						Status: 0,
						Output: "0x6572726F72",
						Hash:   "0x00000000000000000000000000000000000000000000000000000000000000000",
					}, nil,
				},
			},
		})

	ev, err := s.client.ExecuteServiceSync(context.TODO(), service.ExecuteServiceRequest{
		Address: "0x0000000000000000000000000000000000000000",
		Data:    "0x0000000000000000000000000000000000000000",
	})

	assert.Nil(s.T(), err)
	assert.Equal(s.T(), service.ErrorEvent{
		ID: 0,
		Cause: rpc.Error{
			ErrorCode:   1000,
			Description: "transaction receipt has status 0 which indicates a transaction execution failure with error error",
		}}, ev)
}

func (s *ServicesTestSuite) TestGetCodeEmptyAddress() {
	ethtest.ImplementMock(s.ethclient)

	_, err := s.client.GetCode(context.Background(), service.GetCodeRequest{
		Address: "",
	})

	assert.Error(s.T(), err)
	assert.Equal(s.T(), &rpc.Error{ErrorCode: 2006, Description: "Provided invalid address."}, err)
}

func (s *ServicesTestSuite) TestGetCodeOk() {
	ethtest.ImplementMock(s.ethclient)

	res, err := s.client.GetCode(context.Background(), service.GetCodeRequest{
		Address: "0x0000000000000000000000000000000000000000",
	})

	assert.Nil(s.T(), err)
	assert.Equal(s.T(), service.GetCodeResponse{
		Code:    []byte("0x0000000000000000000000000000000000000000"),
		Address: "0x0000000000000000000000000000000000000000",
	}, res)
}

func (s *ServicesTestSuite) TestGetPublicKeyEmptyAddress() {
	ethtest.ImplementMock(s.ethclient)

	_, err := s.client.GetPublicKey(context.Background(), service.GetPublicKeyRequest{
		Address: "",
	})

	assert.Error(s.T(), err)
	assert.Equal(s.T(), &rpc.Error{ErrorCode: 2006, Description: "Provided invalid address."}, err)
}

func (s *ServicesTestSuite) TestGetPublicKeyOk() {
	ethtest.ImplementMock(s.ethclient)

	res, err := s.client.GetPublicKey(context.Background(), service.GetPublicKeyRequest{
		Address: "0x0000000000000000000000000000000000000000",
	})

	assert.Nil(s.T(), err)
	assert.Equal(s.T(), service.GetPublicKeyResponse{
		Timestamp: 1234,
		PublicKey: "0x6f6704e5a10332af6672e50b3d9754dc460dfa4d",
		Signature: "0x6f6704e5a10332af6672e50b3d9754dc460dfa4d",
		Address:   "0x0000000000000000000000000000000000000000",
	}, res)
}

func TestServicesTestSuite(t *testing.T) {
	suite.Run(t, new(ServicesTestSuite))
}
