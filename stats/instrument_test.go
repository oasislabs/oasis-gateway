package stats

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMethodTrackerNoMethods(t *testing.T) {
	tracker := NewMethodTracker()
	assert.Equal(t, []string{"undefined"}, tracker.Methods())

	v, err := tracker.Instrument("something", func() (interface{}, error) {
		return nil, nil
	})
	assert.Nil(t, err)
	assert.Nil(t, v)

	stats := tracker.Stats()
	assert.Equal(t, map[string]interface{}{
		"error":     uint64(0),
		"ok":        uint64(1),
		"undefined": uint64(0),
	}, stats["undefined"].(Metrics)["count"])

	avg := stats["undefined"].(Metrics)["latency"].(Metrics)["avg"].(float64)
	assert.True(t, 0 <= avg && avg <= 1000000)
}

func TestMethodTrackerCountNotFound(t *testing.T) {
	tracker := NewMethodTracker()

	group, ok := tracker.Count("not found")

	assert.Nil(t, group)
	assert.False(t, ok)
}

func TestMethodTrackerCountOK(t *testing.T) {
	tracker := NewMethodTracker()

	group, ok := tracker.Count("undefined")

	assert.NotNil(t, group)
	assert.True(t, ok)
}

func TestMethodTrackerLatenciesNotFound(t *testing.T) {
	tracker := NewMethodTracker()

	group, ok := tracker.Latencies("not found")

	assert.Nil(t, group)
	assert.False(t, ok)
}

func TestMethodTrackerLatenciesOK(t *testing.T) {
	tracker := NewMethodTracker()

	window, ok := tracker.Latencies("undefined")

	assert.NotNil(t, window)
	assert.True(t, ok)
}

func TestMethodTrackerNoMethodsError(t *testing.T) {
	tracker := NewMethodTracker()
	assert.Equal(t, []string{"undefined"}, tracker.Methods())

	v, err := tracker.Instrument("something", func() (interface{}, error) {
		return nil, errors.New("error")
	})
	assert.Error(t, err)
	assert.Nil(t, v)

	stats := tracker.Stats()
	assert.Equal(t, map[string]interface{}{
		"error":     uint64(1),
		"ok":        uint64(0),
		"undefined": uint64(0),
	}, stats["undefined"].(Metrics)["count"])

	avg := stats["undefined"].(Metrics)["latency"].(Metrics)["avg"].(float64)
	assert.True(t, 0 <= avg && avg <= 1000000)
}

func TestMethodTrackerWithMethods(t *testing.T) {
	tracker := NewMethodTracker("method1", "method2")
	assert.ElementsMatch(t, []string{"method1", "method2", "undefined"}, tracker.Methods())

	v, err := tracker.Instrument("method1", func() (interface{}, error) {
		return nil, nil
	})
	assert.Nil(t, err)
	assert.Nil(t, v)

	stats := tracker.Stats()
	assert.Equal(t, map[string]interface{}{
		"error":     uint64(0),
		"ok":        uint64(1),
		"undefined": uint64(0),
	}, stats["method1"].(Metrics)["count"])

	assert.Equal(t, map[string]interface{}{
		"error":     uint64(0),
		"ok":        uint64(0),
		"undefined": uint64(0),
	}, stats["method2"].(Metrics)["count"])

	assert.Equal(t, map[string]interface{}{
		"error":     uint64(0),
		"ok":        uint64(0),
		"undefined": uint64(0),
	}, stats["undefined"].(Metrics)["count"])

	avg := stats["method1"].(Metrics)["latency"].(Metrics)["avg"].(float64)
	assert.True(t, 0 <= avg && avg <= 1000000)
	assert.Equal(t, float64(0),
		stats["method2"].(Metrics)["latency"].(Metrics)["avg"].(float64))
	assert.Equal(t, float64(0),
		stats["undefined"].(Metrics)["latency"].(Metrics)["avg"].(float64))
}

func TestMethodTrackerWithResult(t *testing.T) {
	tracker := NewMethodTrackerWithResult(&MethodTrackerProps{
		Methods:    []string{"method1", "method2"},
		Results:    []string{"result1", "result2"},
		WindowSize: 64,
	})
	assert.ElementsMatch(t, []string{"method1", "method2", "undefined"}, tracker.Methods())

	v, err := tracker.InstrumentResult("method1", func() *TrackResult {
		return &TrackResult{Value: nil, Error: nil, Type: "result2"}
	})
	assert.Nil(t, err)
	assert.Nil(t, v)

	stats := tracker.Stats()
	assert.Equal(t, map[string]interface{}{
		"result1":   uint64(0),
		"result2":   uint64(1),
		"undefined": uint64(0),
	}, stats["method1"].(Metrics)["count"])

	assert.Equal(t, map[string]interface{}{
		"result1":   uint64(0),
		"result2":   uint64(0),
		"undefined": uint64(0),
	}, stats["method2"].(Metrics)["count"])

	assert.Equal(t, map[string]interface{}{
		"result1":   uint64(0),
		"result2":   uint64(0),
		"undefined": uint64(0),
	}, stats["undefined"].(Metrics)["count"])

	avg := stats["method1"].(Metrics)["latency"].(Metrics)["avg"].(float64)
	assert.True(t, 0 <= avg && avg <= 1000000)
	assert.Equal(t, float64(0),
		stats["method2"].(Metrics)["latency"].(Metrics)["avg"].(float64))
	assert.Equal(t, float64(0),
		stats["undefined"].(Metrics)["latency"].(Metrics)["avg"].(float64))
}
