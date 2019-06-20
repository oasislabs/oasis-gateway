package stats

import (
	"time"
)

// ResultTypeBool returns the string representation
// of a bool result used for tracking
func ResultTypeBool(ok bool) string {
	if ok {
		return "ok"
	}

	return "error"
}

// TrackResult is the result of an instrumented
// function
type TrackResult struct {
	// Value of the instrumented function that is passed on
	// to the caller
	Value interface{}

	// Error of the instrumented function that is passed on
	// to the caller
	Error error

	// Type is the representation of the type that is used
	// for instrumentation. It categorizes the result
	Type string
}

// MethodTracker tracks method calls and latencies. It is
// useful to track all the calls within a single type.
// All the information of what a MethodTracker needs to
// track is defined at initialization time to avoid concurrency
// issues with the implementation, so it favours immutability.
// If an unexpected method is tracked the result is stored in
// the special "undefined" category.
type MethodTracker struct {
	count     map[string]*CounterGroup
	latencies map[string]*IntWindow
}

// MethodTrackerProps are the properties used to define
// the behaviour of a MethodTracker
type MethodTrackerProps struct {
	Methods    []string
	Results    []string
	WindowSize uint32
}

// NewMethodTrackerWithResult creates a new MethodTracker with
// the specified properties
func NewMethodTrackerWithResult(props *MethodTrackerProps) *MethodTracker {
	count := make(map[string]*CounterGroup)
	latencies := make(map[string]*IntWindow)

	for _, key := range props.Methods {
		count[key] = NewCounterGroup(props.Results...)
		latencies[key] = NewIntWindow(props.WindowSize)
	}

	count["undefined"] = NewCounterGroup(props.Results...)
	latencies["undefined"] = NewIntWindow(props.WindowSize)

	return &MethodTracker{
		count:     count,
		latencies: latencies,
	}
}

// NewMethodTracker creates a new method tracker using the defaults
// ResultTypeBool for Result types.
func NewMethodTracker(methods ...string) *MethodTracker {
	return NewMethodTrackerWithResult(&MethodTrackerProps{
		Methods:    methods,
		Results:    []string{"ok", "error"},
		WindowSize: 64,
	})
}

// Methods returns the list of methods tracked
func (t *MethodTracker) Methods() []string {
	methods := make([]string, len(t.count))
	for method := range t.count {
		methods = append(methods, method)
	}
	return methods
}

// Count returns the counter used to track the method
// calls. If the method is not found it return nil, false
func (t *MethodTracker) Count(method string) (*CounterGroup, bool) {
	group, ok := t.count[method]
	return group, ok
}

// Latencies returns the window used to track the method
// call latencies. If the method is not found it return nil, false
func (t *MethodTracker) Latencies(method string) (*IntWindow, bool) {
	window, ok := t.latencies[method]
	return window, ok
}

// InstrumentResult instruments the call to a method
// collecting counts and latencies
func (t *MethodTracker) InstrumentResult(
	name string,
	fn func() *TrackResult,
) (interface{}, error) {
	start := time.Now().UnixNano()
	result := fn()
	end := time.Now().UnixNano()
	latency := end - start
	t.StoreLatency(name, latency)
	t.AddCount(name, result.Type)

	return result.Value, result.Error
}

// Instrument instruments the call to a method
// collecting counts and latencies. It uses the defaults
// ResultTypeBool for the collection
func (t *MethodTracker) Instrument(
	name string,
	fn func() (interface{}, error),
) (interface{}, error) {
	return t.InstrumentResult(name, func() *TrackResult {
		v, err := fn()
		return &TrackResult{
			Value: v,
			Error: err,
			Type:  ResultTypeBool(err == nil),
		}
	})
}

// AddCount is a method to manually add a count to
// a method
func (t *MethodTracker) AddCount(name string, result string) {
	group, ok := t.count[name]
	if !ok {
		group = t.count["undefined"]
	}

	group.Incr(result)
}

// StoreLatency is a method to manually store a new latency
// sample for a method
func (t *MethodTracker) StoreLatency(name string, latency int64) {
	l, ok := t.latencies[name]
	if !ok {
		l = t.latencies["undefined"]
	}

	l.Add(latency)
}

// Stats is the implementation of Collector for MethodTracker
func (t *MethodTracker) Stats() Metrics {
	stats := make(Metrics)

	for method, count := range t.count {
		methodStats := make(Metrics)
		methodStats["count"] = count.Stats()
		methodStats["latency"] = t.latencies[method].Stats()
		stats[method] = methodStats
	}

	return stats
}
