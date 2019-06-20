package stats

import (
	"sync/atomic"
)

// Counter is used to count how many times an event
// occurs
type Counter struct {
	value uint64
}

// Increment increments the counter by one
func (c *Counter) Incr() uint64 {
	return atomic.AddUint64(&c.value, 1)
}

// Get returns the current value of the counter
func (c *Counter) Value() uint64 {
	return atomic.AddUint64(&c.value, 0)
}

// CounterGroup implements a group of counters
// where counters can be added dynamically
type CounterGroup struct {
	group map[string]*Counter
}

// NewCounterGroup creates a new counter group. All counters
// are allocated at the creation time.
func NewCounterGroup(names ...string) *CounterGroup {
	m := make(map[string]*Counter)

	for _, name := range names {
		m[name] = &Counter{value: 0}
	}

	// this is the catch all counter for all the requests
	// to increment a counter that does not exist
	m["undefined"] = &Counter{value: 0}

	return &CounterGroup{
		group: m,
	}
}

// Get retrieves the counter from the group. If no counter
// is found associated to that specific name a catch all
// counter is returned
func (g *CounterGroup) Get(name string) *Counter {
	counter, ok := g.group[name]
	if !ok {
		counter = g.group["undefined"]
	}

	return counter
}

// Incr increments the required counter and if it
// does not exist it creates it
func (g *CounterGroup) Incr(name string) uint64 {
	counter := g.Get(name)
	return counter.Incr()
}

// Value implements Collector for CounterGroup
func (g *CounterGroup) Stats() map[string]interface{} {
	stats := make(map[string]interface{})

	for key, counter := range g.group {
		stats[key] = counter.Value()
	}

	return stats
}
