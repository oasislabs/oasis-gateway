package rw

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLimitReaderRead(t *testing.T) {
	buf := bytes.NewBuffer([]byte("some data"))
	p := make([]byte, 64)

	n, err := NewLimitReader(buf, ReadLimitProps{FailOnExceed: true, Limit: 16}).Read(p)

	assert.Nil(t, err)
	assert.Equal(t, 0, buf.Len())
	assert.Equal(t, 9, n)
	assert.Equal(t, "some data", string(p[:n]))
}

func TestLimitReaderReadBufTooSmall(t *testing.T) {
	buf := bytes.NewBuffer([]byte("some data"))
	p := make([]byte, 8)

	r := NewLimitReader(buf, ReadLimitProps{FailOnExceed: false, Limit: 16})

	n, err := r.Read(p)
	assert.Nil(t, err)
	assert.Equal(t, 1, buf.Len())
	assert.Equal(t, 8, n)

	n, err = r.Read(p[n:])
	assert.Nil(t, err)
	assert.Equal(t, 1, buf.Len())
	assert.Equal(t, 0, n)
}

func TestLimitReaderReadWithLimitErrExceed(t *testing.T) {
	buf := bytes.NewBuffer([]byte("some data"))
	p := make([]byte, 64)

	n, err := NewLimitReader(buf, ReadLimitProps{FailOnExceed: true, Limit: 8}).Read(p)

	assert.Equal(t, ErrLimitExceeded, err)
	assert.Equal(t, 0, buf.Len())
	assert.Equal(t, 0, n)
}

func TestLimitReaderReadWithLimit(t *testing.T) {
	buf := bytes.NewBuffer([]byte("some data"))
	p := make([]byte, 64)

	n, err := NewLimitReader(buf, ReadLimitProps{FailOnExceed: false, Limit: 8}).Read(p)

	assert.Nil(t, err)
	assert.Equal(t, 1, buf.Len())
	assert.Equal(t, 8, n)
	assert.Equal(t, "some dat", string(p[:n]))
}

func TestCopyWithLimit(t *testing.T) {
	r := bytes.NewBuffer([]byte("some data"))
	w := bytes.NewBuffer([]byte{})

	n, err := CopyWithLimit(w, r, ReadLimitProps{
		FailOnExceed: false,
		Limit:        16,
	})

	assert.Nil(t, err)
	assert.Equal(t, int64(9), n)
	assert.Equal(t, 0, r.Len())
	assert.Equal(t, 9, w.Len())
	assert.Equal(t, "some data", string(w.Bytes()))
}

func TestCopyWithLimitErrExceed(t *testing.T) {
	r := bytes.NewBuffer([]byte("some data"))
	w := bytes.NewBuffer([]byte{})

	n, err := CopyWithLimit(w, r, ReadLimitProps{
		FailOnExceed: true,
		Limit:        8,
	})

	assert.Equal(t, ErrLimitExceeded, err)
	assert.Equal(t, int64(0), n)
}
