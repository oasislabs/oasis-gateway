package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/oasislabs/developer-gateway/api/v0/service"
	backend "github.com/oasislabs/developer-gateway/backend/core"
	"github.com/oasislabs/developer-gateway/conc"
)

type ID struct {
	ID uint64 `json:"id"`
}

type PollServiceResponseDeserializer struct {
	response service.PollServiceResponse
	Requests map[uint64]backend.EventType
}

type PollServiceResponseDeserialized struct {
	Offset uint64            `json:"offset"`
	Events []json.RawMessage `json:"events"`
}

func (d *PollServiceResponseDeserializer) Deserialize(data []byte) error {
	var res PollServiceResponseDeserialized
	if err := json.Unmarshal(data, &res); err != nil {
		return err
	}

	var events []service.Event
	for _, ev := range res.Events {
		m := ID{}
		if err := json.Unmarshal(ev, &m); err != nil {
			return fmt.Errorf("failed to deserialize json into map %s", err.Error())
		}

		id := m.ID
		t, ok := d.Requests[id]
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
		case t == backend.DeployServiceEventType:
			var res service.DeployServiceEvent
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

func NewServiceClient() ServiceClient {
	return ServiceClient{
		Client:   NewClient(),
		Requests: make(map[uint64]backend.EventType),
	}
}

type ServiceClient struct {
	Client
	Requests map[uint64]backend.EventType
}

func (c ServiceClient) DeployService(
	req service.DeployServiceRequest,
) (service.DeployServiceResponse, error) {
	var res service.DeployServiceResponse
	if err := c.Client.Request(&res, &req, Route{
		Method: "POST",
		Path:   "/v0/api/service/deploy",
	}); err != nil {
		return res, err
	}

	c.Requests[res.ID] = backend.DeployServiceEventType

	return res, nil
}

func (c ServiceClient) PollService(
	req service.PollServiceRequest,
) (service.PollServiceResponse, error) {
	de := PollServiceResponseDeserializer{
		Requests: c.Requests,
	}

	err := c.Client.RequestWithDeserializer(&de, &req, Route{
		Method: "POST",
		Path:   "/v0/api/service/poll",
	})

	return de.response, err
}

func (c ServiceClient) PollServiceUntilNotEmpty(
	req service.PollServiceRequest,
) (service.PollServiceResponse, error) {
	v, err := conc.RetryWithConfig(context.Background(), conc.SupplierFunc(func() (interface{}, error) {
		v, err := c.PollService(req)
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
