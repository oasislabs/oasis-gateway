package rpc

import (
	"bytes"
	"context"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/oasislabs/developer-gateway/log"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

var logger = log.NewLogrus(log.LogrusLoggerProperties{
	Level:  logrus.DebugLevel,
	Output: ioutil.Discard,
})

func mapEntityFactory() EntityFactory {
	return EntityFactoryFunc(func() interface{} { m := make(map[string]string); return &m })
}

func simpleHandlerFactory(factory EntityFactory, handler Handler) HttpMiddleware {
	return HttpMiddlewareRelay{handler: handler}
}

type HandlerEcho struct{}

func (m HandlerEcho) Handle(ctx context.Context, v interface{}) (interface{}, error) {
	return v, nil
}

type HttpMiddlewareRelay struct {
	handler Handler
}

func (m HttpMiddlewareRelay) ServeHTTP(req *http.Request) (interface{}, error) {
	return m.handler.Handle(req.Context(), nil)
}

type HttpMiddlewareOK struct {
	body interface{}
}

func (m HttpMiddlewareOK) ServeHTTP(req *http.Request) (interface{}, error) {
	return m.body, nil
}

type HttpMiddlewarePanic struct{}

func (m HttpMiddlewarePanic) ServeHTTP(req *http.Request) (interface{}, error) {
	panic("error")
}

func setupRouter() *HttpRouter {
	return &HttpRouter{
		encoder: &JsonEncoder{},
		mux: map[string]*HttpRoute{
			"/path": &HttpRoute{handlers: map[string]HttpMiddleware{
				"GET": HttpMiddlewareOK{body: map[string]string{"result": "ok"}},
				"PUT": HttpMiddlewareOK{body: nil},
			}},
			"/panic": &HttpRoute{handlers: map[string]HttpMiddleware{
				"GET": HttpMiddlewarePanic{},
			}},
		},
		logger: logger,
	}
}

func TestHttpRouterServeHTTPNoRoute(t *testing.T) {
	router := setupRouter()

	recorder := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/unknown", nil)

	router.ServeHTTP(recorder, req)

	s, err := ioutil.ReadAll(recorder.Body)

	assert.Nil(t, err)
	assert.Equal(t, http.StatusNotFound, recorder.Code)
	assert.Equal(t, "", string(s))
}

func TestHttpRouterServeHTTPNoMethod(t *testing.T) {
	router := setupRouter()

	recorder := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/path", nil)

	router.ServeHTTP(recorder, req)

	s, err := ioutil.ReadAll(recorder.Body)

	assert.Nil(t, err)
	assert.Equal(t, http.StatusMethodNotAllowed, recorder.Code)
	assert.Equal(t, "", string(s))
}

func TestHttpRouterServeHTTPOKNoBody(t *testing.T) {
	router := setupRouter()

	recorder := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", "/path", nil)

	router.ServeHTTP(recorder, req)

	s, err := ioutil.ReadAll(recorder.Body)

	assert.Nil(t, err)
	assert.Equal(t, http.StatusNoContent, recorder.Code)
	assert.Equal(t, "", string(s))
}

func TestHttpRouterServeHTTPOKWithBody(t *testing.T) {
	router := setupRouter()

	recorder := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/path", nil)

	router.ServeHTTP(recorder, req)

	s, err := ioutil.ReadAll(recorder.Body)

	assert.Nil(t, err)
	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.Equal(t, "{\"result\":\"ok\"}\n", string(s))
}

func TestHttpRouterServeHTTPPanic(t *testing.T) {
	router := setupRouter()

	recorder := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/panic", nil)

	router.ServeHTTP(recorder, req)

	s, err := ioutil.ReadAll(recorder.Body)

	assert.Nil(t, err)
	assert.Equal(t, http.StatusInternalServerError, recorder.Code)
	assert.Equal(t, "{\"errorCode\":1000,\"description\":\"Internal Error. Please check the status of the service.\"}\n", string(s))
}

func TestHttpBinderBuildRouterNoEncoder(t *testing.T) {
	assert.Panics(t, func() {
		NewHttpBinder(HttpBinderProperties{
			Logger:         logger,
			HandlerFactory: HttpHandlerFactoryFunc(simpleHandlerFactory),
		})
	})
}

func TestHttpBinderBuildRouterNoLogger(t *testing.T) {
	assert.Panics(t, func() {
		NewHttpBinder(HttpBinderProperties{
			Encoder:        JsonEncoder{},
			HandlerFactory: HttpHandlerFactoryFunc(simpleHandlerFactory),
		})
	})
}

func TestHttpBinderBuildRouterNoFactory(t *testing.T) {
	assert.Panics(t, func() {
		NewHttpBinder(HttpBinderProperties{
			Encoder: JsonEncoder{},
			Logger:  logger,
		})
	})
}

func TestHttpBinderBuildRouter(t *testing.T) {
	binder := NewHttpBinder(HttpBinderProperties{
		Encoder:        JsonEncoder{},
		Logger:         logger,
		HandlerFactory: HttpHandlerFactoryFunc(simpleHandlerFactory),
	})

	binder.Bind("GET", "/path", HandlerEcho{}, nil)
	router := binder.Build()

	recorder := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/path", nil)

	router.ServeHTTP(recorder, req)

	s, err := ioutil.ReadAll(recorder.Body)

	assert.Nil(t, err)
	assert.Equal(t, http.StatusNoContent, recorder.Code)
	assert.Equal(t, "", string(s))
}

func TestHttpJsonHandlerContentLengthMissing(t *testing.T) {
	handler := NewHttpJsonHandler(HttpJsonHandlerProperties{
		Limit:   1024,
		Handler: HandlerEcho{},
		Logger:  logger,
		Factory: mapEntityFactory(),
	})

	req, _ := http.NewRequest("GET", "/path", nil)
	req.ContentLength = -1

	v, err := handler.ServeHTTP(req)

	assert.Equal(t, "[2002] error code InputError with desc Content-length header missing from request.", err.Error())
	assert.Nil(t, v)
}

func TestHttpJsonHandlerContentLengthMissingWithBody(t *testing.T) {
	handler := NewHttpJsonHandler(HttpJsonHandlerProperties{
		Limit:   1024,
		Handler: HandlerEcho{},
		Logger:  logger,
		Factory: mapEntityFactory(),
	})

	req, _ := http.NewRequest("GET", "/path", bytes.NewBufferString(""))

	v, err := handler.ServeHTTP(req)

	assert.Equal(t, "[2002] error code InputError with desc Content-length header missing from request.", err.Error())
	assert.Nil(t, v)
}

func TestHttpJsonHandlerContentLengthExceedsLimit(t *testing.T) {
	handler := NewHttpJsonHandler(HttpJsonHandlerProperties{
		Limit:   1024,
		Handler: HandlerEcho{},
		Logger:  logger,
		Factory: mapEntityFactory(),
	})

	req, _ := http.NewRequest("GET", "/path", bytes.NewBufferString(""))
	req.ContentLength = 2048

	v, err := handler.ServeHTTP(req)

	assert.Equal(t, "[2003] error code InputError with desc Content-length exceeds request limit.", err.Error())
	assert.Nil(t, v)
}

func TestHttpJsonHandlerContentMissing(t *testing.T) {
	handler := NewHttpJsonHandler(HttpJsonHandlerProperties{
		Limit:   1024,
		Handler: HandlerEcho{},
		Logger:  logger,
		Factory: mapEntityFactory(),
	})

	req, _ := http.NewRequest("GET", "/path",
		bytes.NewBufferString("{\"hamburger\":\"rare\",\"potato\":\"fried\"}\n"))
	req.ContentLength = 38

	v, err := handler.ServeHTTP(req)

	assert.Equal(t, "[2004] error code InputError with desc Content-type should be application/json.", err.Error())
	assert.Nil(t, v)
}

func TestHttpJsonHandlerOK(t *testing.T) {
	handler := NewHttpJsonHandler(HttpJsonHandlerProperties{
		Limit:   1024,
		Handler: HandlerEcho{},
		Logger:  logger,
		Factory: mapEntityFactory(),
	})

	req, _ := http.NewRequest("GET", "/path",
		bytes.NewBufferString("{\"hamburger\":\"rare\",\"potato\":\"fried\"}\n"))
	req.ContentLength = 38
	req.Header.Add("Content-type", "application/json")

	v, err := handler.ServeHTTP(req)
	m := *v.(*map[string]string)
	assert.Nil(t, err)
	assert.Equal(t, map[string]string{"hamburger": "rare", "potato": "fried"}, m)
}
