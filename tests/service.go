package tests

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/oasislabs/developer-gateway/api/v0/service"
	"github.com/oasislabs/developer-gateway/conc"
)

type ServiceClient struct {
	Client
}

func (c ServiceClient) DeployService(
	req service.DeployServiceRequest,
) (service.DeployServiceResponse, error) {
	var res service.DeployServiceResponse
	err := c.Client.Request(&res, &req, Route{
		Method: "POST",
		Path:   "/v0/api/service/deploy",
	})
	return res, err
}

func (c ServiceClient) PollService(
	req service.PollServiceRequest,
) (service.PollServiceResponse, error) {
	var res service.PollServiceResponse
	err := c.Client.Request(&res, &req, Route{
		Method: "POST",
		Path:   "/v0/api/service/poll",
	})
	return res, err
}

func (c ServiceClient) PollServiceUntilNotEmpty(
	req service.PollServiceRequest,
) (service.PollServiceResponse, error) {
	v, err := conc.RetryWithConfig(context.Background(), conc.SupplierFunc(func() (interface{}, error) {
		fmt.Println("ATTEMPT")
		v, err := c.PollService(req)
		if err != nil {
			fmt.Println("ERR: ", err, err == nil)
			return nil, conc.ErrCannotRecover{Cause: err}
		}

		if len(v.Events) == 0 {
			return nil, errors.New("no events yet")
		}

		return v, nil
	}), conc.RetryConfig{
		Random:            false,
		UnlimitedAttempts: false,
		Attempts:          10,
		BaseExp:           2,
		BaseTimeout:       1 * time.Millisecond,
		MaxRetryTimeout:   100 * time.Millisecond,
	})

	if err != nil {
		return service.PollServiceResponse{}, err
	}
	return v.(service.PollServiceResponse), nil
}
