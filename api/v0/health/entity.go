package health

import "github.com/oasislabs/developer-gateway/stats"

// GetHealthRequest is a request to retrieve the health
// status of the component.
type GetHealthRequest struct{}

// GetHealthResponse is the response to the health request
type GetHealthResponse struct {
	Health  stats.HealthStatus `json:"health"`
	Metrics stats.Metrics      `json:"metrics"`
}
