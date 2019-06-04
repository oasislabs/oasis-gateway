package tests

import (
	"context"
	"testing"

	"github.com/oasislabs/developer-gateway/api/v0/service"
	"github.com/oasislabs/developer-gateway/rpc"
	"github.com/oasislabs/developer-gateway/tests/apitest"
	"github.com/oasislabs/developer-gateway/tests/mock"
	"github.com/stretchr/testify/assert"
)

func TestDeployServiceEmptyData(t *testing.T) {
	client := apitest.NewServiceClient(router)
	_, err := client.DeployService(context.Background(), service.DeployServiceRequest{
		Data: "",
	})

	assert.Equal(t, &rpc.Error{ErrorCode: 7002, Description: "Failed to verify AAD in transaction data."}, err)
}

func TestDeployServiceErr(t *testing.T) {
	client := apitest.NewServiceClient(router)
	deployRes, err := client.DeployService(context.Background(), service.DeployServiceRequest{
		Data: mock.TransactionDataErr,
	})
	assert.Nil(t, err)
	assert.Equal(t, uint64(0), deployRes.ID)

	pollRes, err := client.PollServiceUntilNotEmpty(context.Background(), service.PollServiceRequest{
		Offset: 0,
	})

	assert.Nil(t, err)
	assert.Equal(t, uint64(0), pollRes.Offset)
	assert.Equal(t, 1, len(pollRes.Events))
	assert.Equal(t, service.ErrorEvent{
		ID: 0x0,
		Cause: rpc.Error{
			ErrorCode:   1002,
			Description: "Internal Error. Please check the status of the service.",
		}}, pollRes.Events[0])
}

func TestDeployServiceOK(t *testing.T) {
	client := apitest.NewServiceClient(router)
	deployRes, err := client.DeployService(context.Background(), service.DeployServiceRequest{
		Data: mock.TransactionDataOK,
	})
	assert.Nil(t, err)
	assert.Equal(t, uint64(0), deployRes.ID)

	pollRes, err := client.PollServiceUntilNotEmpty(context.Background(), service.PollServiceRequest{
		Offset: deployRes.ID,
	})

	assert.Nil(t, err)
	assert.Equal(t, uint64(0), pollRes.Offset)
	assert.Equal(t, 1, len(pollRes.Events))
	assert.Equal(t, service.DeployServiceEvent{
		ID:      0,
		Address: "0x0000000000000000000000000000000000000000",
	}, pollRes.Events[0])
}

func TestExecuteServiceEmptyAddress(t *testing.T) {
	client := apitest.NewServiceClient(router)
	_, err := client.ExecuteService(context.Background(), service.ExecuteServiceRequest{
		Address: "",
		Data:    mock.TransactionDataOK,
	})

	assert.Error(t, err)
	assert.Equal(t, &rpc.Error{ErrorCode: 2006, Description: "Provided invalid address."}, err)
}

func TestExecuteServiceEmptyTransactionData(t *testing.T) {
	client := apitest.NewServiceClient(router)
	_, err := client.ExecuteService(context.Background(), service.ExecuteServiceRequest{
		Address: mock.Address,
		Data:    "",
	})

	assert.Error(t, err)
	assert.Equal(t, &rpc.Error{ErrorCode: 7002, Description: "Failed to verify AAD in transaction data."}, err)
}

func TestExecuteServiceErr(t *testing.T) {
	client := apitest.NewServiceClient(router)
	executeRes, err := client.ExecuteService(context.Background(), service.ExecuteServiceRequest{
		Address: mock.Address,
		Data:    mock.TransactionDataErr,
	})
	assert.Nil(t, err)
	assert.Equal(t, uint64(0), executeRes.ID)

	pollRes, err := client.PollServiceUntilNotEmpty(context.Background(), service.PollServiceRequest{
		Offset: 0,
	})

	assert.Nil(t, err)
	assert.Equal(t, uint64(0), pollRes.Offset)
	assert.Equal(t, 1, len(pollRes.Events))
	assert.Equal(t, service.ErrorEvent{
		ID: 0x0,
		Cause: rpc.Error{
			ErrorCode:   1002,
			Description: "Internal Error. Please check the status of the service.",
		}}, pollRes.Events[0])
}

func TestExecuteServiceOK(t *testing.T) {
	client := apitest.NewServiceClient(router)
	executeRes, err := client.ExecuteService(context.Background(), service.ExecuteServiceRequest{
		Address: mock.Address,
		Data:    mock.TransactionDataOK,
	})
	assert.Nil(t, err)
	assert.Equal(t, uint64(0), executeRes.ID)

	pollRes, err := client.PollServiceUntilNotEmpty(context.Background(), service.PollServiceRequest{
		Offset: 0,
	})

	assert.Nil(t, err)
	assert.Equal(t, uint64(0), pollRes.Offset)
	assert.Equal(t, 1, len(pollRes.Events))
	assert.Equal(t, service.ExecuteServiceEvent{
		ID:      0,
		Address: "0x0000000000000000000000000000000000000000",
		Output:  "0x00",
	}, pollRes.Events[0])
}

func TestGetPublicKeyEmptyAddress(t *testing.T) {
	client := apitest.NewServiceClient(router)
	_, err := client.GetPublicKey(context.Background(), service.GetPublicKeyRequest{
		Address: "",
	})

	assert.Error(t, err)
	assert.Equal(t, &rpc.Error{ErrorCode: 2006, Description: "Provided invalid address."}, err)
}

func TestGetPublicKeyOk(t *testing.T) {
	client := apitest.NewServiceClient(router)
	res, err := client.GetPublicKey(context.Background(), service.GetPublicKeyRequest{
		Address: mock.Address,
	})

	assert.Nil(t, err)
	assert.Equal(t, service.GetPublicKeyResponse{
		Timestamp: 0x1b69b4bab46a831,
		Address:   "0x0000000000000000000000000000000000000000",
		PublicKey: "0x0000000000000000000000000000000000000000",
		Signature: "0x0000000000000000000000000000000000000000",
	}, res)
}
