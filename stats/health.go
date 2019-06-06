package stats

// HealthStatus of a service. This status should be advertised by
// a service so that a health checker can know what action
// if any is required to keep the status on a Healthy state
type HealthStatus uint

const (
	// Healthy status means that the service is up and running
	// and can take incoming requests
	Healthy HealthStatus = 0

	// Drain a service so that it processes the requests that
	// already are inflight but does not take further requests
	Drain HealthStatus = 1

	// Unhealthy status for a service that should be restarted
	Unhealthy HealthStatus = 2
)
