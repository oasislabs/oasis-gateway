package tests

import (
	"testing"

	"github.com/oasislabs/developer-gateway/api/v0/service"
	"github.com/oasislabs/developer-gateway/rpc"
	"github.com/stretchr/testify/assert"
)

func TestDeployServiceEmptyData(t *testing.T) {
	client := NewServiceClient()
	_, err := client.DeployService(service.DeployServiceRequest{
		Data: "",
	})

	assert.Equal(t, &rpc.Error{ErrorCode: 7002, Description: "Failed to verify AAD in transaction data."}, err)
}

func TestDeployServiceErr(t *testing.T) {
	client := NewServiceClient()
	deployRes, err := client.DeployService(service.DeployServiceRequest{
		Data: TransactionDataErr,
	})
	assert.Nil(t, err)
	assert.Equal(t, uint64(0), deployRes.ID)

	pollRes, err := client.PollServiceUntilNotEmpty(service.PollServiceRequest{
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
	client := NewServiceClient()
	deployRes, err := client.DeployService(service.DeployServiceRequest{
		Data: TransactionDataOK,
	})
	assert.Nil(t, err)
	assert.Equal(t, uint64(0), deployRes.ID)

	pollRes, err := client.PollServiceUntilNotEmpty(service.PollServiceRequest{
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
	client := NewServiceClient()
	_, err := client.ExecuteService(service.ExecuteServiceRequest{
		Address: "",
		Data:    TransactionDataOK,
	})

	assert.Error(t, err)
	assert.Equal(t, &rpc.Error{ErrorCode: 2006, Description: "Provided invalid address."}, err)
}

func TestExecuteServiceEmptyTransactionData(t *testing.T) {
	client := NewServiceClient()
	_, err := client.ExecuteService(service.ExecuteServiceRequest{
		Address: Address,
		Data:    "",
	})

	assert.Error(t, err)
	assert.Equal(t, &rpc.Error{ErrorCode: 7002, Description: "Failed to verify AAD in transaction data."}, err)
}

func TestExecuteServiceErr(t *testing.T) {
	client := NewServiceClient()
	executeRes, err := client.ExecuteService(service.ExecuteServiceRequest{
		Address: Address,
		Data:    TransactionDataErr,
	})
	assert.Nil(t, err)
	assert.Equal(t, uint64(0), executeRes.ID)

	pollRes, err := client.PollServiceUntilNotEmpty(service.PollServiceRequest{
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
	client := NewServiceClient()
	executeRes, err := client.ExecuteService(service.ExecuteServiceRequest{
		Address: Address,
		Data:    TransactionDataOK,
	})
	assert.Nil(t, err)
	assert.Equal(t, uint64(0), executeRes.ID)

	pollRes, err := client.PollServiceUntilNotEmpty(service.PollServiceRequest{
		Offset: 0,
	})

	assert.Nil(t, err)
	assert.Equal(t, uint64(0), pollRes.Offset)
	assert.Equal(t, 1, len(pollRes.Events))
	assert.Equal(t, service.ExecuteServiceEvent{
		ID:      0,
		Address: "0x0000000000000000000000000000000000000000",
	}, pollRes.Events[0])
}

func TestGetPublicKeyServiceEmptyAddress(t *testing.T) {
	client := NewServiceClient()
	_, err := client.GetPublicKey(service.GetPublicKeyServiceRequest{
		Address: "",
	})

	assert.Error(t, err)
	assert.Equal(t, &rpc.Error{ErrorCode: 2007, Description: "Input cannot be empty."}, err)
}

func TestGetPublicKeyServiceOk(t *testing.T) {
	client := NewServiceClient()
	res, err := client.GetPublicKey(service.GetPublicKeyServiceRequest{
		Address: Address,
	})

	assert.Nil(t, err)
	assert.Equal(t, service.GetPublicKeyServiceResponse{
		Timestamp: 0x1b69b4bab46a831,
		Address:   "0x0000000000000000000000000000000000000000",
		PublicKey: "0x0000000000000000000000000000000000000000",
		Signature: "0x0000000000000000000000000000000000000000",
	}, res)
}
