package gateway

import (
	"runtime"

	"github.com/oasislabs/oasis-gateway/stats"
)

// RuntimeService is an abstraction of the go runtime
// to be able to expose application metrics
type RuntimeService struct{}

// Name is the implementation of Service.Name
// for RuntimeService
func (s RuntimeService) Name() string {
	return "runtime"
}

// Stats is the implementation of Service.Stats
// for RuntimeService
func (s RuntimeService) Stats() stats.Metrics {
	metrics := make(stats.Metrics)
	metrics["NumCPU"] = runtime.NumCPU()
	metrics["NumGoroutine"] = runtime.NumGoroutine()
	return metrics
}
