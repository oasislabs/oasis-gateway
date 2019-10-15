package health

import (
	"context"

	"github.com/oasislabs/oasis-gateway/rpc"
	"github.com/oasislabs/oasis-gateway/stats"
)

type Deps struct {
	Collector stats.Collector
}

type HealthHandler struct {
	collector stats.Collector
}

func NewHealthHandler(deps *Deps) HealthHandler {
	return HealthHandler{collector: deps.Collector}
}

func (h HealthHandler) GetHealth(ctx context.Context, v interface{}) (interface{}, error) {
	_ = v.(*GetHealthRequest)
	return &GetHealthResponse{
		Health:  stats.Healthy,
		Metrics: h.collector.Stats(),
	}, nil
}

func BindHandler(deps *Deps, binder rpc.HandlerBinder) {
	handler := NewHealthHandler(deps)

	binder.Bind("GET", "/v0/api/health", rpc.HandlerFunc(handler.GetHealth),
		rpc.EntityFactoryFunc(func() interface{} { return &GetHealthRequest{} }))
}
