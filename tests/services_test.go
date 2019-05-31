package tests

import (
	"fmt"
	"testing"

	"github.com/oasislabs/developer-gateway/api/v0/service"
	"github.com/oasislabs/developer-gateway/rpc"
	"github.com/stretchr/testify/assert"
)

func TestDeployServiceEmptyData(t *testing.T) {
	_, err := ServiceClient{}.DeployService(service.DeployServiceRequest{
		Data: "",
	})

	assert.Equal(t, &rpc.Error{ErrorCode: 2007, Description: "Input cannot be empty."}, err)
}

func TestDeployService(t *testing.T) {
	client := ServiceClient{}
	deployRes, err := client.DeployService(service.DeployServiceRequest{
		Data: "0x98503a1f1275ddfc8621778e5477aed91290df0cbf2e7796363e9aa4c4d9b0c130f27c3bf55264e9ffe9c01d90ca216e5161171ed51878fd8993c16410a87d59642ad75eeac7a24ca018a42811684264bedb517f59e71ab2d27a38086bd0f4c47d39c255",
	})
	assert.Nil(t, err)
	assert.Equal(t, uint64(0), deployRes.ID)

	fmt.Println("ATTEMPTING")
	pollRes, err := client.PollServiceUntilNotEmpty(service.PollServiceRequest{
		Offset: deployRes.ID,
	})

	fmt.Println(pollRes)
	assert.Nil(t, err)
	assert.Nil(t, pollRes)
}
