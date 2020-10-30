package metrics

import (
	"fmt"

	"github.com/prometheus/client_golang/prometheus"
)

var (
	// Labels to use for partitioning requests.
	requestLabels = []string{"endpoint", "status", "cause"}

	// Labels to use for partitioning request latencies.
	requestLatencyLabels = []string{"endpoint"}
)

// Default service metrics for data sovereignty services.
type ServiceMetrics struct {
	// Counts of requests made to each service endpoint.
	Requests *prometheus.CounterVec

	// Latencies of request transactions for each request operation the calling service supports.
	RequestLatencies *prometheus.SummaryVec
}

// NewDefaultServiceMetrics creates Prometheus metric instrumentation for basic
// metrics common to typical services. Default metrics include:
//
// 1. Counts of service endpoints hit.
// 2. Latencies for requests.
func NewDefaultServiceMetrics(serviceName string) *ServiceMetrics {
	metrics := &ServiceMetrics{
		Requests: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: fmt.Sprintf("%s_requests", serviceName),
				Help: fmt.Sprintf("How many service requests were made, partitioned by request endpoint, status, and cause of failure."),
			},
			requestLabels,
		),
		RequestLatencies: prometheus.NewSummaryVec(
			prometheus.SummaryOpts{
				Name: fmt.Sprintf("%s_request_durations", serviceName),
				Help: fmt.Sprintf("How long requests take to process, partitioned by the request endpoint."),
			},
			requestLatencyLabels,
		),
	}
	prometheus.MustRegister(metrics.RequestLatencies)
	prometheus.MustRegister(metrics.Requests)
	return metrics
}

// RequestCounter returns the counter for the calling request.
// Provided labels should be endpoint, status, cause.
func (m *ServiceMetrics) RequestCounter(labels ...string) prometheus.Counter {
	if len(labels) > len(requestLabels) {
		labels = labels[:len(requestLabels)]
	}
	labels = append(labels, make([]string, len(requestLabels)-len(labels))...)
	return m.Requests.WithLabelValues(labels...)
}

// RequestTimer creates a new latency timer for the provided request operation.
func (m *ServiceMetrics) RequestTimer(labels ...string) *prometheus.Timer {
	if len(labels) > len(requestLatencyLabels) {
		labels = labels[:len(requestLatencyLabels)]
	}
	labels = append(labels, make([]string, len(requestLatencyLabels)-len(labels))...)
	return prometheus.NewTimer(m.RequestLatencies.WithLabelValues(labels...))
}
