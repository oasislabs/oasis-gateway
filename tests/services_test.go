package tests

import (
	"context"
	"testing"

	"github.com/oasislabs/developer-gateway/api/v0/service"
	"github.com/oasislabs/developer-gateway/gateway"
	"github.com/oasislabs/developer-gateway/rpc"
	"github.com/oasislabs/developer-gateway/tests/apitest"
	"github.com/oasislabs/developer-gateway/tests/mock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type ServicesTestSuite struct {
	suite.Suite
	client *apitest.ServiceClient
}

func (s *ServicesTestSuite) SetupTest() {
	services, err := mock.NewServices(context.TODO(), Config)
	if err != nil {
		panic(err)
	}

	router := gateway.NewPublicRouter(services)
	s.client = apitest.NewServiceClient(router)
}

func (s *ServicesTestSuite) TestDeployServiceEmptyData() {
	_, err := s.client.DeployService(context.Background(), service.DeployServiceRequest{
		Data: "",
	})

	assert.Equal(s.T(), &rpc.Error{ErrorCode: 7002, Description: "Failed to verify AAD in transaction data."}, err)
}

func (s *ServicesTestSuite) TestDeployServiceErr() {
	deployRes, err := s.client.DeployService(context.Background(), service.DeployServiceRequest{
		Data: mock.TransactionDataErr,
	})

	assert.Nil(s.T(), err)
	assert.Equal(s.T(), uint64(0), deployRes.ID)

	pollRes, err := s.client.PollServiceUntilNotEmpty(context.Background(), service.PollServiceRequest{
		Offset: 0,
	})

	assert.Nil(s.T(), err)
	assert.Equal(s.T(), uint64(0), pollRes.Offset)
	assert.Equal(s.T(), 1, len(pollRes.Events))
	assert.Equal(s.T(), service.ErrorEvent{
		ID: 0x0,
		Cause: rpc.Error{
			ErrorCode:   1002,
			Description: "Internal Error. Please check the status of the service.",
		}}, pollRes.Events[0])
}

func (s *ServicesTestSuite) TestDeployServiceOK() {
	deployRes, err := s.client.DeployService(context.Background(), service.DeployServiceRequest{
		Data: mock.TransactionDataOK,
	})
	assert.Nil(s.T(), err)
	assert.Equal(s.T(), uint64(0), deployRes.ID)

	pollRes, err := s.client.PollServiceUntilNotEmpty(context.Background(), service.PollServiceRequest{
		Offset: deployRes.ID,
	})

	assert.Nil(s.T(), err)
	assert.Equal(s.T(), uint64(0), pollRes.Offset)
	assert.Equal(s.T(), 1, len(pollRes.Events))
	assert.Equal(s.T(), service.DeployServiceEvent{
		ID:      0,
		Address: "0x0000000000000000000000000000000000000000",
	}, pollRes.Events[0])
}

func (s *ServicesTestSuite) TestExecuteServiceEmptyAddress() {
	_, err := s.client.ExecuteService(context.Background(), service.ExecuteServiceRequest{
		Address: "",
		Data:    mock.TransactionDataOK,
	})

	assert.Error(s.T(), err)
	assert.Equal(s.T(), &rpc.Error{ErrorCode: 2006, Description: "Provided invalid address."}, err)
}

func (s *ServicesTestSuite) TestExecuteServiceEmptyTransactionData() {
	_, err := s.client.ExecuteService(context.Background(), service.ExecuteServiceRequest{
		Address: mock.Address,
		Data:    "",
	})

	assert.Error(s.T(), err)
	assert.Equal(s.T(), &rpc.Error{ErrorCode: 7002, Description: "Failed to verify AAD in transaction data."}, err)
}

func (s *ServicesTestSuite) TestExecuteServiceErr() {
	executeRes, err := s.client.ExecuteService(context.Background(), service.ExecuteServiceRequest{
		Address: mock.Address,
		Data:    mock.TransactionDataErr,
	})
	assert.Nil(s.T(), err)
	assert.Equal(s.T(), uint64(0), executeRes.ID)

	pollRes, err := s.client.PollServiceUntilNotEmpty(context.Background(), service.PollServiceRequest{
		Offset: 0,
	})

	assert.Nil(s.T(), err)
	assert.Equal(s.T(), uint64(0), pollRes.Offset)
	assert.Equal(s.T(), 1, len(pollRes.Events))
	assert.Equal(s.T(), service.ErrorEvent{
		ID: 0x0,
		Cause: rpc.Error{
			ErrorCode:   1002,
			Description: "Internal Error. Please check the status of the service.",
		}}, pollRes.Events[0])
}

func (s *ServicesTestSuite) TestExecuteServiceOK() {
	executeRes, err := s.client.ExecuteService(context.Background(), service.ExecuteServiceRequest{
		Address: mock.Address,
		Data:    mock.TransactionDataOK,
	})
	assert.Nil(s.T(), err)
	assert.Equal(s.T(), uint64(0), executeRes.ID)

	pollRes, err := s.client.PollServiceUntilNotEmpty(context.Background(), service.PollServiceRequest{
		Offset: 0,
	})

	assert.Nil(s.T(), err)
	assert.Equal(s.T(), uint64(0), pollRes.Offset)
	assert.Equal(s.T(), 1, len(pollRes.Events))
	assert.Equal(s.T(), service.ExecuteServiceEvent{
		ID:      0,
		Address: "0x0000000000000000000000000000000000000000",
		Output:  "0x73756363657373",
	}, pollRes.Events[0])
}

func (s *ServicesTestSuite) TestExecuteServiceReceiptStatusErr() {
	executeRes, err := s.client.ExecuteService(context.Background(), service.ExecuteServiceRequest{
		Address: mock.Address,
		Data:    mock.TransactionDataReceiptErr,
	})
	assert.Nil(s.T(), err)
	assert.Equal(s.T(), uint64(0), executeRes.ID)

	pollRes, err := s.client.PollServiceUntilNotEmpty(context.Background(), service.PollServiceRequest{
		Offset: 0,
	})
	assert.Nil(s.T(), err)
	assert.Equal(s.T(), 1, len(pollRes.Events))
	assert.Equal(s.T(), service.ErrorEvent{
		ID: 0,
		Cause: rpc.Error{
			ErrorCode:   1000,
			Description: "transaction receipt has status 0 which indicates a transaction execution failure with error error",
		}}, pollRes.Events[0])
}

func (s *ServicesTestSuite) TestGetPublicKeyEmptyAddress() {
	_, err := s.client.GetPublicKey(context.Background(), service.GetPublicKeyRequest{
		Address: "",
	})

	assert.Error(s.T(), err)
	assert.Equal(s.T(), &rpc.Error{ErrorCode: 2006, Description: "Provided invalid address."}, err)
}

func (s *ServicesTestSuite) TestGetPublicKeyOk() {
	res, err := s.client.GetPublicKey(context.Background(), service.GetPublicKeyRequest{
		Address: mock.Address,
	})

	assert.Nil(s.T(), err)
	assert.Equal(s.T(), service.GetPublicKeyResponse{
		Timestamp: 0x1b69b4bab46a831,
		Address:   "0x0000000000000000000000000000000000000000",
		PublicKey: "0x0000000000000000000000000000000000000000",
		Signature: "0x0000000000000000000000000000000000000000",
	}, res)
}

func TestServicesTestSuite(t *testing.T) {
	suite.Run(t, new(ServicesTestSuite))
}
