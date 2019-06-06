package health

import (
	"context"

	"github.com/oasislabs/developer-gateway/rpc"
	"github.com/oasislabs/developer-gateway/stats"
)

type Services struct{}

type HealthHandler struct{}

func NewHealthHandler(services Services) HealthHandler {
	return HealthHandler{}
}

func (h HealthHandler) GetHealth(ctx context.Context, v interface{}) (interface{}, error) {
	_ = v.(*GetHealthRequest)
	return &GetHealthResponse{Health: stats.Healthy}, nil
}

func BindHandler(services Services, binder rpc.HandlerBinder) {
	handler := NewHealthHandler(services)

	binder.Bind("GET", "/v0/api/health", rpc.HandlerFunc(handler.GetHealth),
		rpc.EntityFactoryFunc(func() interface{} { return &GetHealthRequest{} }))
}
