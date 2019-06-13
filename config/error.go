package config

import (
	"errors"
	"fmt"
	"strings"
)

type ErrNotImplemented struct {
	Key   string
	Value string
}

func (e ErrNotImplemented) Error() string {
	return fmt.Sprintf("configuration key %s does not have option %s implemented",
		e.Key, e.Value)
}

type ErrKeyNotSet struct {
	Key string
}

func (e ErrKeyNotSet) Error() string {
	return fmt.Sprintf("configuration key needs to be set %s", e.Key)
}

type ErrInvalidValue struct {
	Key          string
	InvalidValue string
	Values       []string
}

func (e ErrInvalidValue) Error() string {
	return fmt.Sprintf("configuration key %s set to invalid value %s. "+
		"Accepted values are: %s.", e.Key, e.InvalidValue, strings.Join(e.Values, ", "))
}

type ErrParseFlags struct {
	Cause error
}

func (e ErrParseFlags) Error() string {
	return fmt.Sprintf("failed to parse flags %s", e.Cause.Error())
}

var (
	ErrAlreadyParsed error = errors.New("arguments already parsed")
)
