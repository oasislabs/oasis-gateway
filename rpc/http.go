package rpc

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/oasislabs/developer-gateway/log"
)

const HttpHeaderTraceID string = "OASIS-TRACE-ID"

type HttpMiddleware interface {
	ServeHTTP(req *http.Request) (interface{}, error)
}

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
	handlers map[string]HttpMiddleware
	encoder  Encoder
	logger   log.Logger
}

// HttpRoute implementation of HttpMiddleware
func (h HttpRoute) ServeHTTP(req *http.Request) (interface{}, error) {
	handler, ok := h.handlers[req.Method]
	if !ok {
		return nil, HttpNotFound(req.Context(), "")
	}

	return handler.ServeHTTP(req)
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
	path := req.URL.EscapedPath()
	method := req.Method
	traceID := ParseTraceID(req.Header.Get(HttpHeaderTraceID))
	req.WithContext(context.WithValue(req.Context(), log.ContextKeyTraceID, traceID))

	h.logger.Debug(req.Context(), "", log.MapFields{
		"path":      path,
		"method":    method,
		"call_type": "HttpRequestHandleAttempt",
	})

	defer func() {
		if r := recover(); r != nil {
			var err error

			switch x := r.(type) {
			case string:
				err = errors.New(x)
				h.logger.Warn(req.Context(), "unexpected panic caught", log.MapFields{
					"path":      path,
					"method":    method,
					"call_type": "HttpRequestHandleFailure",
					"err":       err,
				})
			case error:
				err = x
				h.logger.Warn(req.Context(), "unexpected panic caught", log.MapFields{
					"path":      path,
					"method":    method,
					"call_type": "HttpRequestHandleFailure",
					"err":       err.Error(),
				})
			default:
				err = fmt.Errorf("unknown panic %+v", r)
				h.logger.Warn(req.Context(), "unexpected panic caught", log.MapFields{
					"path":      path,
					"method":    method,
					"call_type": "HttpRequestHandleFailure",
					"err":       err,
				})
			}

			h.reportAnyError(res, req, errors.New("Unexpected error occurred."))
		}
	}()

	route, ok := h.mux[path]
	if !ok {
		h.reportError(res, req, HttpNotFound(req.Context(), ""))
		return
	}

	v, err := route.ServeHTTP(req)
	if err != nil {
		h.reportAnyError(res, req, err)
		return
	}

	h.reportSuccess(res, req, v)
}

func (h *HttpRouter) reportSuccess(res http.ResponseWriter, req *http.Request, body interface{}) {
	path := req.URL.EscapedPath()
	method := req.Method

	if body == nil {
		res.WriteHeader(http.StatusNoContent)
		h.logger.Info(req.Context(), "", log.MapFields{
			"path":        path,
			"method":      method,
			"call_type":   "HttpRequestHandleSuccess",
			"status_code": http.StatusNoContent,
		})
		return
	}

	if err := h.encoder.Encode(res, body); err != nil {
		h.logger.Warn(req.Context(), "failed to encode response to response writer", log.MapFields{
			"path":        path,
			"method":      method,
			"call_type":   "HttpRequestHandleFailure",
			"status_code": http.StatusNoContent,
		})
		return
	}

	h.logger.Info(req.Context(), "", log.MapFields{
		"path":        path,
		"method":      method,
		"call_type":   "HttpRequestHandleSuccess",
		"status_code": http.StatusOK,
	})
}

func (h *HttpRouter) reportAnyError(res http.ResponseWriter, req *http.Request, err error) {
	switch err := err.(type) {
	case HttpError:
		h.reportError(res, req, &err)
	case Error:
		h.reportError(res, req, &HttpError{
			Cause:      err,
			StatusCode: http.StatusInternalServerError,
		})
	default:
		h.reportError(res, req, HttpInternalServerError(req.Context(), err.Error()))
	}
}

func (h *HttpRouter) reportError(res http.ResponseWriter, req *http.Request, err *HttpError) {
	path := req.URL.EscapedPath()
	method := req.Method

	if err := h.encoder.Encode(res, err); err != nil {
		h.logger.Debug(req.Context(), "failed to encode error response to response writer", log.MapFields{
			"path":      path,
			"method":    method,
			"call_type": "HttpRequestHandleFailure",
			"err":       err.Error(),
		})
	}

	h.logger.Info(req.Context(), "", log.MapFields{
		"path":        path,
		"method":      method,
		"call_type":   "HttpRequestHandleFailure",
		"status_code": err.StatusCode,
		"err":         err.Error(),
	})
}

// HttpJsonHandler is handlers requests that expect a body in the JSON format,
// handles the body and executes the final handler with the expected type
type HttpJsonHandler struct {
	limit   uint
	decoder JsonDecoder
	handler Handler
	logger  log.Logger
	factory Factory
}

// ServeHTTP is the implementation of HttpMiddleware for HttpJsonHandler
func (h *HttpJsonHandler) ServeHTTP(req *http.Request) (interface{}, error) {
	// verify that content length is set and it is correct
	if req.ContentLength < -1 || (req.ContentLength == 0 && req.Body != nil) {
		h.logger.Debug(req.Context(), "Content-length header missing from request", log.MapFields{
			"path":      req.URL.EscapedPath(),
			"method":    req.Method,
			"call_type": "HttpJsonRequestHandleFailure",
		})
		return nil, HttpBadRequest(req.Context(), "")
	}

	if uint64(req.ContentLength) > uint64(h.limit) {
		h.logger.Debug(req.Context(), "Content-length exceeds request limit", log.MapFields{
			"path":          req.URL.EscapedPath(),
			"method":        req.Method,
			"contentLength": req.ContentLength,
			"limit":         h.limit,
			"call_type":     "HttpJsonRequestHandleFailure",
		})
		return nil, HttpBadRequest(req.Context(), "")
	}

	// verify that content type is set and it is correct
	contentType := req.Header.Get("Content-type")
	if contentType != "application/json" {
		h.logger.Debug(req.Context(), "Content-type is not for json", log.MapFields{
			"path":          req.URL.EscapedPath(),
			"method":        req.Method,
			"contentLength": req.ContentLength,
			"call_type":     "HttpJsonRequestHandleFailure",
		})
		return nil, HttpBadRequest(req.Context(), "")
	}

	// parse body into Go object
	body := h.factory.Create()
	if body == nil && req.ContentLength > 0 {
		h.logger.Debug(req.Context(), "received request body for handler that does not expect a request body", log.MapFields{
			"path":          req.URL.EscapedPath(),
			"method":        req.Method,
			"contentLength": req.ContentLength,
			"call_type":     "HttpJsonRequestHandleFailure",
		})
		return nil, HttpBadRequest(req.Context(), "")

	}

	if body != nil {
		if err := h.decoder.DecodeWithLimit(req.Body, body, uint(req.ContentLength)); err != nil {
			h.logger.Debug(req.Context(), "failed to decode json", log.MapFields{
				"path":          req.URL.EscapedPath(),
				"method":        req.Method,
				"contentLength": req.ContentLength,
				"call_type":     "HttpJsonRequestHandleFailure",
				"err":           err.Error(),
			})
			return nil, HttpBadRequest(req.Context(), "")
		}
	}

	// provide the parsed body to the handler and handle execution
	return h.handler.Handle(req.Context(), body)
}
