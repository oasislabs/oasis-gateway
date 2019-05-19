package rw

import (
	"errors"
	"io"
)

// UniRead defines an interface for unidirectional communication
// between a remote endpoint to the local endpoint
type UniRead interface {
	// Read processes input from the remote endpoint provided
	// in the reader and writes to the writer the generated
	// output for the remote endpoint as a response, if any
	Read(io.Writer, io.Reader) (int, error)
}

// UniWrite defines an interface for unidirectional communication
// from the local endpoint to a remote endpoint
type UniWrite interface {
	// Write processes input from the local endpoint provided
	// in the reader and writes to the writer the generated
	// output for the remote endpoint as a response, if any
	Write(io.Writer, io.Reader) (int, error)
}

// UniReadFunc is the implementation of UniRead for functions
type UniReadFunc func(io.Writer, io.Reader) (int, error)

// Read is the implementation of Read for UniRead
func (f UniReadFunc) Read(w io.Writer, r io.Reader) (int, error) {
	return f(w, r)
}

// UniWriteFunc is the implementation of UniWrite for functions
type UniWriteFunc func(io.Writer, io.Reader) (int, error)

// Write is the implementation of Write for UniWrite
func (f UniWriteFunc) Write(w io.Writer, r io.Reader) (int, error) {
	return f(w, r)
}

// BiReadWriter interface for wrappers that need to process
// bidirectional communication between two endpoints.
type BiReadWriter interface {
	UniRead
	UniWrite
}

// ErrLimitExceeded signals that the underlying reader has more
// available bytes than the expected limit
var ErrLimitExceeded = errors.New("Read limit exceeded")

// ReadLimitProps sets up the behaviour of the limit reader
type ReadLimitProps struct {
	// FailOnExceed defines whether the LimitReader should return an
	// error if the underlying reader has more bytes than the limit
	FailOnExceed bool

	// Limit is the maximum number of bytes that can be read from the
	// reader and copied to the provided buffer
	Limit int64
}

// NewLimitReader returns a new LimitReader
func NewLimitReader(reader io.Reader, props ReadLimitProps) LimitReader {
	readerLimit := props.Limit
	if props.FailOnExceed {
		// set io.LimitedReader's limit to props.Limit + 1 to allow it to
		// read one more byte than required. This is the only way we can verify
		// whether the reader has more data to provide than the limit
		readerLimit += 1
	}

	return LimitReader{
		failOnExceed: props.FailOnExceed,
		count:        0,
		limit:        props.Limit,
		reader:       io.LimitReader(reader, readerLimit),
	}
}

// LimitReader is an io.Reader wrapper that ensures that
// no more than limit bytes are read from the reader
type LimitReader struct {
	failOnExceed bool
	count        int64
	limit        int64
	reader       io.Reader
}

// Read is the implementation of Reader for LimitReader
func (r LimitReader) Read(p []byte) (int, error) {
	n, err := r.reader.Read(p)
	if err != nil {
		return 0, err
	}

	r.count += int64(n)
	if r.failOnExceed && r.count > r.limit {
		return 0, ErrLimitExceeded
	}

	return n, nil
}

// CopyWithLimit copies props.Limit bytes from an io.Reader to an io.Writer.
func CopyWithLimit(w io.Writer, r io.Reader, props ReadLimitProps) (int64, error) {
	if r == nil {
		return 0, nil
	}

	if w == nil {
		return 0, errors.New("writer cannot be nil")
	}

	readerLimit := props.Limit
	if props.FailOnExceed {
		// set io.LimitedReader's limit to props.Limit + 1 to allow it to
		// read one more byte than required. This is the only way we can verify
		// whether the reader has more data to provide than the limit
		readerLimit += 1
	}

	n, err := io.CopyN(w, r, readerLimit)
	if err != nil {
		if err == io.EOF {
			if n > props.Limit {
				return 0, ErrLimitExceeded
			}

			return n, nil
		}

		return 0, err
	}

	if n > props.Limit {
		return 0, ErrLimitExceeded
	}

	return n, nil
}
