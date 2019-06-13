package mqueue

import (
	"errors"
	"fmt"
)

var (
	ErrBackendConfigConflict error = errors.New("backend conflict between provider and configuration")
)

type ErrUnknownBackend struct {
	Backend string
}

func (e ErrUnknownBackend) Error() string {
	return fmt.Sprintf("unknown mailbox backend provided: %s", e.Backend)
}
