package core

import "strings"

type MultiError struct {
	Errors []error
}

func (e MultiError) Error() string {
	var s []string
	for _, err := range e.Errors {
		s = append(s, err.Error())
	}

	return strings.Join(s, "; ")
}
