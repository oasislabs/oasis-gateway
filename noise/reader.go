package noise

import (
	"errors"
	"io"
)

func readWithAppendLimit(reader io.Reader, p []byte, limit int) ([]byte, int, error) {
	var bytesRead int
	currentLimit := len(p)

	if currentLimit > limit {
		currentLimit = limit
	}

	for limit > 0 {
		if bytesRead >= currentLimit {
			if bytesRead > currentLimit {
				panic("read more bytes from io.Reader than currentLimit")
			}
			if currentLimit >= limit {
				if currentLimit > limit {
					panic("read more bytes from io.Reader than limit")
				}

				// verify that the reader still has data available, in which case
				// should return with an error
				s := []byte{0}
				n, err := reader.Read(s)
				if err == io.EOF || n == 0 {
					break
				}

				if err != nil {
					return nil, 0, err
				}

				return nil, 0, errors.New("reader has more data than limit data")
			}

			currentLimit = currentLimit << 1
			if currentLimit < 64 {
				currentLimit = 64
			}
			if limit < currentLimit {
				currentLimit = limit
			}

			newP := make([]byte, currentLimit)
			copy(newP, p)
			p = newP
		}

		n, err := reader.Read(p[bytesRead:currentLimit])
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, 0, err
		}

		if n == 0 {
			break
		}

		bytesRead += n
	}

	return p[:bytesRead], bytesRead, nil
}
