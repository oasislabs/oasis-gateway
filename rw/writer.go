package rw

// Writer is a wrapper around a []byte so that
// it can be used as an io.Writer
type Writer struct {
	buf    []byte
	offset int
}

// NewWriter creates a new instance of a Writer
func NewWriter(p []byte) *Writer {
	return &Writer{buf: p, offset: 0}
}

// Write implements io.Write for a Writer
func (w *Writer) Write(p []byte) (int, error) {
	n := copy(w.buf[:w.offset], p)
	w.offset += n
	return n, nil
}
