package apitest

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/google/uuid"
	"github.com/oasislabs/developer-gateway/api/v0/service"
	"github.com/oasislabs/developer-gateway/conc"
	"github.com/oasislabs/developer-gateway/rpc"
)

// ServiceClient is the client implementation for the
// Service API
type ServiceClient struct {
	client   *Client
	requests map[uint64]service.Request
	session  string
}

// NewServiceClient creates a new instance of a service client
// with an underlying client and session ready to be used
// to execute a router API
func NewServiceClient(router *rpc.HttpRouter) *ServiceClient {
	return &ServiceClient{
		client:   NewClient(router),
		requests: make(map[uint64]service.Request),
		session:  uuid.New().String(),
	}
}

// DeployService deploys the specific service
func (c *ServiceClient) DeployService(
	ctx context.Context,
	req service.DeployServiceRequest,
) (service.DeployServiceResponse, error) {
	var res service.DeployServiceResponse
	if err := c.client.RequestAPI(&rpc.SimpleJsonDeserializer{
		O: &res,
	}, &req, c.session, Route{
		Method: "POST",
		Path:   "/v0/api/service/deploy",
	}); err != nil {
		return res, err
	}

	c.requests[res.ID] = &req

	return res, nil
}

// ExecuteService deploys the specific service
func (c *ServiceClient) ExecuteService(
	ctx context.Context,
	req service.ExecuteServiceRequest,
) (service.ExecuteServiceResponse, error) {
	var res service.ExecuteServiceResponse
	if err := c.client.RequestAPI(&rpc.SimpleJsonDeserializer{
		O: &res,
	}, &req, c.session, Route{
		Method: "POST",
		Path:   "/v0/api/service/execute",
	}); err != nil {
		return res, err
	}

	c.requests[res.ID] = &req

	return res, nil
}

func (c *ServiceClient) PollService(
	ctx context.Context,
	req service.PollServiceRequest,
) (service.PollServiceResponse, error) {
	de := PollServiceResponseDeserializer{
		Requests: c.requests,
	}

	err := c.client.RequestAPI(&de, &req, c.session, Route{
		Method: "POST",
		Path:   "/v0/api/service/poll",
	})

	return de.response, err
}

func (c *ServiceClient) GetPublicKey(
	ctx context.Context,
	req service.GetPublicKeyServiceRequest,
) (service.GetPublicKeyServiceResponse, error) {
	var res service.GetPublicKeyServiceResponse
	err := c.client.RequestAPI(&rpc.SimpleJsonDeserializer{
		O: &res,
	}, &req, c.session, Route{
		Method: "GET",
		Path:   "/v0/api/service/getPublicKey",
	})

	return res, err
}

func (c ServiceClient) PollServiceUntilNotEmpty(
	ctx context.Context,
	req service.PollServiceRequest,
) (service.PollServiceResponse, error) {
	v, err := conc.RetryWithConfig(context.Background(), conc.SupplierFunc(func() (interface{}, error) {
		v, err := c.PollService(ctx, req)
		if err != nil {
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

type ID struct {
	ID uint64 `json:"id"`
}

type PollServiceResponseDeserializer struct {
	response service.PollServiceResponse
	Requests map[uint64]service.Request
}

type PollServiceResponseDeserialized struct {
	Offset uint64            `json:"offset"`
	Events []json.RawMessage `json:"events"`
}

func (d *PollServiceResponseDeserializer) Deserialize(r io.Reader) error {
	var res PollServiceResponseDeserialized
	if err := json.NewDecoder(r).Decode(&res); err != nil {
		return err
	}

	var events []service.Event
	for _, ev := range res.Events {
		m := ID{}
		if err := json.Unmarshal(ev, &m); err != nil {
			return fmt.Errorf("failed to deserialize json into map %s", err.Error())
		}

		id := m.ID
		r, ok := d.Requests[id]
		if !ok {
			return errors.New("received event for which ID is not tracked")
		}

		switch {
		case bytes.Contains(ev, []byte("\"errorCode\"")):
			var errEvent service.ErrorEvent
			if err := json.Unmarshal(ev, &errEvent); err != nil {
				return err
			}

			events = append(events, errEvent)
			delete(d.Requests, id)
		case r.Type() == service.Deploy:
			var res service.DeployServiceEvent
			if err := json.Unmarshal(ev, &res); err != nil {
				return err
			}

			events = append(events, res)
			delete(d.Requests, id)

		case r.Type() == service.Execute:
			var res service.ExecuteServiceEvent
			if err := json.Unmarshal(ev, &res); err != nil {
				return err
			}

			events = append(events, res)
			delete(d.Requests, id)

		default:
			panic("received unexpected event type")
		}
	}

	d.response.Offset = res.Offset
	d.response.Events = events
	return nil
}
