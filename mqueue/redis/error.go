package redis

import (
	"errors"
	"fmt"
)

var (
	ErrQueueNotFound = errors.New("queue not found")
	ErrOpNotOk       = errors.New("operation did not return OK")
)

type ErrSerialize struct {
	Cause error
}

func (e ErrSerialize) Error() string {
	return fmt.Sprintf("serialization error  %s", e.Cause)
}

func IsErrSerialize(err error) bool {
	_, ok := err.(ErrSerialize)
	return ok
}

type ErrRedisExec struct {
	Cause error
}

func (e ErrRedisExec) Error() string {
	return fmt.Sprintf("redis exec error %s", e.Cause)
}

func IsErrRedisExec(err error) bool {
	_, ok := err.(ErrRedisExec)
	return ok
}
