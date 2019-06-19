package stats

// IntWindow keeps a window of data based on the number
// of data it can hold. When the window is full, it discards
// the oldest data to leave space for new data
type IntWindow struct {
	offset     uint32
	end        uint32
	maxSamples uint32
	window     []int64
}

// NewWindow creates a new window that samples at most
// maxSamples
func NewIntWindow(maxSamples uint32) *IntWindow {
	return &IntWindow{
		offset:     0,
		end:        0,
		maxSamples: maxSamples,
		window:     make([]int64, maxSamples<<1),
	}
}

// Add a new sample to the window, shifting the window
// if maxSamples has been exceeded
func (w *IntWindow) Add(sample int64) {
	w.window[w.end] = sample
	w.end++

	winlen := len(w.window)
	if w.end == uint32(winlen) {
		// once w.end gets to the end of the window, copy
		// w.maxSamples to the beginning of the window
		// and reset the indices
		index := uint32(winlen) - w.maxSamples
		copy(w.window, w.window[index:])
		w.offset = 0
		w.end = w.maxSamples
	}

	// if the window exceeds the number of max samples
	// allowed, recalculate the indices
	if w.end-w.offset > w.maxSamples {
		w.offset = w.end - w.maxSamples
	}
}

// Stats is the implementation of Collector for IntWindow
func (w *IntWindow) Stats() map[string]interface{} {
	return map[string]interface{}{
		"avg": IntAverage(w.window[w.offset:w.end]),
	}
}
