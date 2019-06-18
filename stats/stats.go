package stats

// Stats is the interface that all types that need to
// provide some sort of statistic information implement
type Stats interface {
	// Value returns the current value of the stats
	// as a string
	Value() string
}

// Stats is a group of related stats that will be
// presented together
type Metrics map[string]Stats

// NewMetrics returns a new instance of Metrics
func NewMetrics() Metrics {
	return Metrics(make(map[string]Stats))
}

// Group groups a set of stats together
type Group map[string]Metrics

// NewGroup returns a new Group of metrics
func NewGroup() Group {
	return Group(make(map[string]Metrics))
}

// Add adds a set of metrics to the group
func (g Group) Add(key string, metrics Metrics) {
	g[key] = metrics
}
