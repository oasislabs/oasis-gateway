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

type ErrTemplateErr struct {
	Cause error
	Param string
}

func (e ErrTemplateErr) Error() string {
	return fmt.Sprintf("[callback] failed to generate %s for http request: %s",
		e.Param, e.Cause.Error())
}
