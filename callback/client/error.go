package client

import "fmt"

type ErrNewHttpRequest struct {
	Cause error
}

func (e ErrNewHttpRequest) Error() string {
	return fmt.Sprintf("[callback] failed to create http request: %s", e.Cause.Error())
}

type ErrDeliverHttpRequest struct {
	Cause error
}

func (e ErrDeliverHttpRequest) Error() string {
	return fmt.Sprintf("[callback] failed to deliver http request: %s", e.Cause.Error())
}
