package backend

import (
	"errors"
	"fmt"
)

var (
	ErrEkidenBackendNotImplemented = errors.New("ekiden backend is not implemented")
)

type ErrUnknownBackend struct {
	Backend string
}

func (e ErrUnknownBackend) Error() string {
	return fmt.Sprintf("unknown backend provided: %s", e.Backend)
}
