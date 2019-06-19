package stats

type Metrics map[string]interface{}

// Collector is the interface for all types that
// generate an aggregation of statistic information implement
type Collector interface {
	Stats() Metrics
}

// IntAverage calculates the average of a slice. Returns
// 0 if slice is empty
func IntAverage(arr []int64) float64 {
	var sum int64

	for _, v := range arr {
		sum += v
	}

	if len(arr) > 0 {
		return float64(sum) / float64(len(arr))
	}

	return 0
}
