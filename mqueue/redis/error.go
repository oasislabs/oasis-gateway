package redis

import (
	"errors"
	"fmt"
)

var (
	ErrScriptNotFound = errors.New("script not found")
	ErrQueueNotFound  = errors.New("queue not found")
	ErrOpNotOk        = errors.New("operation did not return OK")
)

type ErrScriptLoad struct {
	Cause error
}

func (e ErrScriptLoad) Error() string {
	return fmt.Sprintf("script load error  %s", e.Cause)
}

func IsErrScriptLoad(err error) bool {
	_, ok := err.(ErrScriptLoad)
	return ok
}

type ErrDeserialize struct {
	Cause error
}

func (e ErrDeserialize) Error() string {
	return fmt.Sprintf("deserialization error  %s", e.Cause)
}

func IsErrDeserialize(err error) bool {
	_, ok := err.(ErrDeserialize)
	return ok
}

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
