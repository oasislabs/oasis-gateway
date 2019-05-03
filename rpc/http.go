package rpc

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/oasislabs/developer-gateway/log"
)

const HttpHeaderTraceID = "X-OASIS-TRACE-ID"

// HttpMiddleware are the handlers that offer extra functionality to a request and
// that in success will forward the request to another handler
type HttpMiddleware interface {
	// ServeHTTP allows to handle an http request. The response will be serialized
	// by an HttpRouter
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
	if e.Cause == nil {
		return fmt.Sprintf("http error with status code %d", e.StatusCode)
	} else {
		return fmt.Sprintf("%s with status code %d", e.Cause.Error(), e.StatusCode)
	}
}

// MakeHttpError makes a new http error
func MakeHttpError(ctx context.Context, description string, statusCode int) *HttpError {
	return &HttpError{
		Cause: &Error{
			ErrorCode:   -1,
			Description: description,
		},
		StatusCode: statusCode,
	}
}

// HttpBadRequest returns an HTTP bad request error
func HttpBadRequest(ctx context.Context, description string) *HttpError {
	return MakeHttpError(ctx, description, http.StatusBadRequest)
}

// HttpForbidden returns an HTTP not found error
func HttpForbidden(ctx context.Context, description string) *HttpError {
	return MakeHttpError(ctx, description, http.StatusForbidden)
}

// HttpNotFound returns an HTTP not found error
func HttpNotFound(ctx context.Context, description string) *HttpError {
	return MakeHttpError(ctx, description, http.StatusNotFound)
}

// HttpNotFound returns an HTTP not found error
func HttpNotImplemented(ctx context.Context, description string) *HttpError {
	return MakeHttpError(ctx, description, http.StatusNotImplemented)
}

// HttpInternalServerError returns an HTTP internal server error
func HttpInternalServerError(ctx context.Context, description string) *HttpError {
	return MakeHttpError(ctx, description, http.StatusInternalServerError)
}

// HttpRoute multiplexes the handling of a request to the handler
// that expects a particular method
type HttpRoute struct {
	handlers map[string]HttpMiddleware
}

// NewHttpRoute creates a new route instance
func NewHttpRoute() *HttpRoute {
	return &HttpRoute{
		handlers: make(map[string]HttpMiddleware),
	}
}

// HttpRoute implementation of HttpMiddleware
func (h HttpRoute) ServeHTTP(req *http.Request) (interface{}, error) {
	handler, ok := h.handlers[req.Method]
	if !ok {
		return nil, &HttpError{StatusCode: http.StatusMethodNotAllowed}
	}

	return handler.ServeHTTP(req)
}

// HttpRouter multiplexes the handling of server request amongst the different
// handlers
type HttpRouter struct {
	encoder Encoder
	mux     map[string]*HttpRoute
	logger  log.Logger
}

// HttpRouter implementation of http.Handler
func (h *HttpRouter) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	path := req.URL.EscapedPath()
	method := req.Method
	traceID := ParseTraceID(req.Header.Get(HttpHeaderTraceID))
	req = req.WithContext(context.WithValue(req.Context(), log.ContextKeyTraceID, traceID))

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
			case error:
				err = x
			default:
				err = fmt.Errorf("unknown panic %+v", r)
			}

			h.logger.Warn(req.Context(), "unexpected panic caught", log.MapFields{
				"path":      path,
				"method":    method,
				"call_type": "HttpRequestHandleFailure",
				"err":       err,
			})

			// the `err` generated above is an internal error that should not
			// be exposed to the client. Instead, this we just return a generic
			// error
			h.reportAnyError(res, req, errors.New("Unexpected error occurred."))
		}
	}()

	route, ok := h.mux[path]
	if !ok {
		h.reportError(res, req, &HttpError{StatusCode: http.StatusNotFound})
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

	res.Header().Add(HttpHeaderTraceID, strconv.FormatInt(log.GetTraceID(req.Context()), 10))

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
	case *HttpError:
		h.reportError(res, req, err)
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

	res.Header().Add(HttpHeaderTraceID, strconv.FormatInt(log.GetTraceID(req.Context()), 10))
	res.WriteHeader(err.StatusCode)

	if err.Cause != nil {
		if err := h.encoder.Encode(res, err.Cause); err != nil {
			h.logger.Debug(req.Context(), "failed to encode error response to response writer", log.MapFields{
				"path":      path,
				"method":    method,
				"call_type": "HttpRequestHandleFailure",
				"err":       err.Error(),
			})
			return
		}
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
	factory EntityFactory
}

type HttpJsonHandlerProperties struct {
	// Limit is the maximum number of bytes an Http body can have. Bodies
	// with a higher limit will fail to deserialize and be rejected
	Limit uint

	// Handler is the rpc handler that will be used to handle the request
	Handler Handler

	// Logger
	Logger log.Logger

	// Factory for creating new instances of objects to which the Http body
	// will be deserialized. Those instances will be passed to the handler
	Factory EntityFactory
}

// NewHttpJsonHandlerFactory creates a new instance of an rpc handler
// that deserializes json objects into Go objects
func NewHttpJsonHandler(properties HttpJsonHandlerProperties) *HttpJsonHandler {
	limit := properties.Limit

	// set a reasonable default limit in case Limit is not set
	if limit == 0 {
		limit = 1 << 14 // 16 KB
	}

	if properties.Handler == nil {
		panic("handler must be sett")
	}

	if properties.Logger == nil {
		panic("logger must be set")
	}

	if properties.Factory == nil {
		panic("factory must be set")
	}

	return &HttpJsonHandler{
		limit:   limit,
		decoder: JsonDecoder{},
		handler: properties.Handler,
		logger:  properties.Logger.ForClass("http", "HttpJsonHandler"),
		factory: properties.Factory,
	}
}

// ServeHTTP is the implementation of HttpMiddleware for HttpJsonHandler
func (h *HttpJsonHandler) ServeHTTP(req *http.Request) (interface{}, error) {
	// verify that content length is set and it is correct
	if req.ContentLength < 0 || (req.ContentLength == 0 && req.Body != nil) {
		h.logger.Debug(req.Context(), "Content-length header missing from request", log.MapFields{
			"path":      req.URL.EscapedPath(),
			"method":    req.Method,
			"call_type": "HttpJsonRequestHandleFailure",
		})
		return nil, HttpBadRequest(req.Context(), "Content-length header missing from request")
	}

	if uint64(req.ContentLength) > uint64(h.limit) {
		h.logger.Debug(req.Context(), "Content-length exceeds request limit", log.MapFields{
			"path":           req.URL.EscapedPath(),
			"method":         req.Method,
			"content_length": req.ContentLength,
			"limit":          h.limit,
			"call_type":      "HttpJsonRequestHandleFailure",
		})
		return nil, HttpBadRequest(req.Context(), "Content-length exceeds request limit")
	}

	// verify that content type is set and it is correct
	contentType := req.Header.Get("Content-type")
	if contentType != "application/json" {
		h.logger.Debug(req.Context(), "Content-type is not for json", log.MapFields{
			"path":           req.URL.EscapedPath(),
			"method":         req.Method,
			"content_length": req.ContentLength,
			"call_type":      "HttpJsonRequestHandleFailure",
		})
		return nil, HttpBadRequest(req.Context(), "Content-type should be application/json")
	}

	// parse body into Go object
	body := h.factory.Create()
	if body == nil && req.ContentLength > 0 {
		h.logger.Debug(req.Context(), "received request body for handler that does not expect a request body", log.MapFields{
			"path":           req.URL.EscapedPath(),
			"method":         req.Method,
			"content_length": req.ContentLength,
			"call_type":      "HttpJsonRequestHandleFailure",
		})
		return nil, HttpBadRequest(req.Context(), "Failed to deserialize request body as JSON")

	}

	if body != nil {
		if err := h.decoder.DecodeWithLimit(req.Body, body, uint(req.ContentLength)); err != nil {
			h.logger.Debug(req.Context(), "failed to decode json", log.MapFields{
				"path":           req.URL.EscapedPath(),
				"method":         req.Method,
				"content_length": req.ContentLength,
				"call_type":      "HttpJsonRequestHandleFailure",
				"err":            err.Error(),
			})
			return nil, HttpBadRequest(req.Context(), "")
		}
	}

	// provide the parsed body to the handler and handle execution
	return h.handler.Handle(req.Context(), body)
}

// HttpHandlerFactory converts an rpc Handler into HttpMiddleware
// that can be plugged into a router
type HttpHandlerFactory interface {
	Make(factory EntityFactory, handler Handler) HttpMiddleware
}

// HttpHandlerFactoryFunc to allow functions to act as an HttpHandlerFactory
type HttpHandlerFactoryFunc func(factory EntityFactory, handler Handler) HttpMiddleware

// Make is the implementation of HttpHandlerFactory for HttpHandlerFactoryFunc
func (f HttpHandlerFactoryFunc) Make(factory EntityFactory, handler Handler) HttpMiddleware {
	return f(factory, handler)
}

// HttpBinder is the binder for http. It is also the only mechanism to build
// HttpRouter's. This is done so that an HttpRouter cannot be modified
// after it has been created
type HttpBinder struct {
	handlers map[string]*HttpRoute
	encoder  Encoder
	logger   log.Logger
	factory  HttpHandlerFactory
}

// Bind is the implementation of HandlerBinder for HttpBinder
func (b *HttpBinder) Bind(method string, uri string, handler Handler, factory EntityFactory) {
	route, ok := b.handlers[uri]
	if !ok {
		route = NewHttpRoute()
		b.handlers[uri] = route
	}

	route.handlers[method] = b.factory.Make(factory, handler)
}

// Build creates a new HttpRouter and clears the handler map of the
// HttpBinder, so if new instances of HttpRouters need to be build
// Bind needs to be used again
func (b *HttpBinder) Build() *HttpRouter {
	handlers := b.handlers

	// avoid modification of the router handlers after the router
	// handler has been created
	b.handlers = make(map[string]*HttpRoute)

	return &HttpRouter{
		encoder: b.encoder,
		logger:  b.logger.ForClass("http", "router"),
		mux:     handlers,
	}
}

// HttpBinderProperties are the properties used to create
// a new instance of an HttpBinder
type HttpBinderProperties struct {
	Encoder        Encoder
	Logger         log.Logger
	HandlerFactory HttpHandlerFactory
}

// NewHttpBinder creates a new instance of the HttpBinder. It will
// panic in case there are errors in the construction of the binder
func NewHttpBinder(properties HttpBinderProperties) *HttpBinder {
	if properties.Encoder == nil {
		panic("Encoder must be set")
	}

	if properties.Logger == nil {
		panic("Logger must be set")
	}

	if properties.HandlerFactory == nil {
		panic("HandlerFactory must be set")
	}

	return &HttpBinder{
		handlers: make(map[string]*HttpRoute),
		encoder:  properties.Encoder,
		logger:   properties.Logger,
		factory:  properties.HandlerFactory,
	}
}
