package rpc

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/oasislabs/developer-gateway/log"
)

const HttpHeaderTraceID string = "OASIS-TRACE-ID"

// HttpError holds the necessary information to return an error when
// using the http protocol
type HttpError struct {
	// Cause of the creation of this HttpError instance
	Cause error

	// StatusCode is the HTTP status code that defines the error cause
	StatusCode int
}

// Error is the implementation of go's error interface for Error
func (e HttpError) Error() string {
	return fmt.Sprintf("%s with status code %d", e.Cause.Error(), e.StatusCode)
}

// MakeHttpError makes a new http error
func MakeHttpError(ctx context.Context, description string, statusCode int) *HttpError {
	return &HttpError{
		Cause: &Error{
			TraceID:     log.GetTraceID(ctx),
			ErrorCode:   -1,
			Description: description,
		},
		StatusCode: statusCode,
	}
}

// BadRequest returns an HTTP bad request
func HttpBadRequest(ctx context.Context, description string) *HttpError {
	return MakeHttpError(ctx, description, http.StatusBadRequest)
}

// NotFound returns an HTTP bad request
func HttpNotFound(ctx context.Context, description string) *HttpError {
	return MakeHttpError(ctx, description, http.StatusNotFound)
}

// NotFound returns an HTTP bad request
func HttpInternalServerError(ctx context.Context, description string) *HttpError {
	return MakeHttpError(ctx, description, http.StatusInternalServerError)
}

// HttpRoute multiplexes the handling of a request to the handler
// that expects a particular method
type HttpRoute struct {
	handlers map[string]http.Handler
	encoder  Encoder
	logger   log.Logger
}

// HttpRoute implementation of http.Handler
func (h HttpRoute) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	handler, ok := h.handlers[req.Method]
	if !ok {
		h.reportError(req.Context(), res, HttpNotFound(req.Context(), ""))
		return
	}

	handler.ServeHTTP(res, req)
}

func (h HttpRoute) reportError(ctx context.Context, res http.ResponseWriter, err *HttpError) {
	if err := h.encoder.Encode(res, err); err != nil {
		h.logger.Debug(ctx, "failed to encode error response to response writer", log.MapFields{
			"call_type": "WriteErrorFailure",
			"err":       err.Error(),
		})
	}
}

// HttpRouter multiplexes the handling of server request amongst the different
// handlers
type HttpRouter struct {
	encoder Encoder
	mux     map[string]HttpRoute
	logger  log.Logger
}

// HttpRouter implementation of http.Handler
func (h *HttpRouter) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	traceID := ParseTraceID(req.Header.Get(HttpHeaderTraceID))
	req.WithContext(context.WithValue(req.Context(), log.ContextKeyTraceID, traceID))

	defer func() {
		if r := recover(); r != nil {
			var err error

			switch x := r.(type) {
			case string:
				err = errors.New(x)
				h.logger.Warn(req.Context(), "unexpected panic caught", log.MapFields{
					"call_type": "PanicCaught",
					"err":       err,
				})
			case error:
				err = x
				h.logger.Warn(req.Context(), "unexpected panic caught", log.MapFields{
					"call_type": "PanicCaught",
					"err":       err.Error(),
				})
			default:
				err = fmt.Errorf("unknown panic %+v", r)
				h.logger.Warn(req.Context(), "unexpected panic caught", log.MapFields{
					"call_type": "PanicCaught",
					"err":       err,
				})
			}

			h.reportAnyError(req.Context(), res, errors.New("Unexpected error occurred."))
		}
	}()

	path := req.URL.EscapedPath()
	route, ok := h.mux[path]
	if !ok {
		h.reportError(req.Context(), res, HttpNotFound(req.Context(), ""))
		return
	}

	route.ServeHTTP(res, req)
}

func (h *HttpRouter) reportAnyError(ctx context.Context, res http.ResponseWriter, err error) {
	switch err := err.(type) {
	case HttpError:
		h.reportError(ctx, res, &err)
	case Error:
		h.reportError(ctx, res, &HttpError{
			Cause:      err,
			StatusCode: http.StatusInternalServerError,
		})
	default:
		h.reportError(ctx, res, HttpInternalServerError(ctx, err.Error()))
	}
}

func (h *HttpRouter) reportError(ctx context.Context, res http.ResponseWriter, err *HttpError) {
	if err := h.encoder.Encode(res, err); err != nil {
		h.logger.Debug(ctx, "failed to encode error response to response writer", log.MapFields{
			"call_type": "WriteErrorFailure",
			"err":       err.Error(),
		})
	}
}
