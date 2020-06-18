package metrics

import (
	"fmt"

	"github.com/prometheus/client_golang/prometheus"
)

var (
	// Labels to use for database operations.
	operationLabels = []string{"operation", "status", "cause"}

	// Labels to use for database latencies.
	operationLatencyLabels = []string{"operation"}
)

// Default service metrics for data sovereignty services.
type DatabaseMetrics struct {
	// Counts of redis operations.
	DatabaseOperations *prometheus.CounterVec

	// Latencies of redis operations.
	DatabaseLatencies *prometheus.SummaryVec
}

// NewDefaultDatabaseMetrics creates Prometheus metric instrumentation for basic
// metrics common to typical services. Default metrics include:
//
// 1. Counts of service endpoints hit.
// 2. Latencies for requests.
func NewDefaultDatabaseMetrics(serviceName string) *DatabaseMetrics {
	metrics := &DatabaseMetrics{
		DatabaseOperations: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: fmt.Sprintf("redis_requests"),
				Help: fmt.Sprintf("How many redis operations are made, partitioned by operation"),
			},
			operationLabels,
		),
		DatabaseLatencies: prometheus.NewSummaryVec(
			prometheus.SummaryOpts{
				Name: fmt.Sprintf("redis_latencies"),
				Help: fmt.Sprintf("How long redis operations take, partitioned by operation"),
			},
			operationLatencyLabels,
		),
	}
	prometheus.MustRegister(metrics.DatabaseOperations)
	prometheus.MustRegister(metrics.DatabaseLatencies)
	return metrics
}

// DatabaseCounterreturns the counter for the database operation.
// Provided labels should be operation, status, and cause.
func (m *DatabaseMetrics) DatabaseCounter(labels ...string) prometheus.Counter {
	if len(labels) > len(operationLabels) {
		labels = labels[:len(operationLabels)]
	}
	labels = append(labels, make([]string, len(operationLabels)-len(labels))...)
	return m.DatabaseOperations.WithLabelValues(labels...)
}

// DatabaseTimer creates a new latency timer for the provided database operation.
func (m *DatabaseMetrics) DatabaseTimer(labels ...string) *prometheus.Timer {
	if len(labels) > len(operationLatencyLabels) {
		labels = labels[:len(operationLatencyLabels)]
	}
	labels = append(labels, make([]string, len(operationLatencyLabels)-len(labels))...)
	return prometheus.NewTimer(m.DatabaseLatencies.WithLabelValues(labels...))
}
