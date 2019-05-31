package tests

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"

	"github.com/google/uuid"
	auth "github.com/oasislabs/developer-gateway/auth/core"
	"github.com/oasislabs/developer-gateway/auth/insecure"
	"github.com/oasislabs/developer-gateway/rpc"
)

type SimpleDeserializer struct {
	O interface{}
}

func (d *SimpleDeserializer) Deserialize(data []byte) error {
	return json.Unmarshal(data, d.O)
}

type Deserializer interface {
	Deserialize([]byte) error
}

type Request struct {
	Route   Route
	Body    []byte
	Headers map[string]string
}

type Response struct {
	Code int
	Body []byte
}

type Route struct {
	Method string
	Path   string
}

func NewClient() Client {
	return Client{Session: uuid.New().String()}
}

type Client struct {
	Session string
}

func (c Client) RequestWithDeserializer(de Deserializer, req interface{}, route Route) error {
	p, err := json.Marshal(req)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal request body %s", err.Error()))
	}

	result, err := ServeHTTP(router, Request{
		Route: route,
		Body:  p,
		Headers: map[string]string{
			insecure.HeaderKey:           "mykey",
			auth.RequestHeaderSessionKey: c.Session,
			"Content-type":               "application/json",
		},
	})
	if err != nil {
		return err
	}

	if result.Code != http.StatusOK {
		var rpcError rpc.Error
		if err := json.Unmarshal(result.Body, &rpcError); err != nil {
			panic(fmt.Sprintf("failed to unmarshal response body as error %s", err.Error()))
		}

		return &rpcError
	}

	if err := de.Deserialize(result.Body); err != nil {
		panic(fmt.Sprintf("failed to unmarshal response body %s", err.Error()))
	}

	return nil
}

func (c Client) Request(res interface{}, req interface{}, route Route) error {
	return c.RequestWithDeserializer(&SimpleDeserializer{O: res}, req, route)
}

func ServeHTTP(router *rpc.HttpRouter, request Request) (Response, error) {
	var (
		req *http.Request
		err error
	)
	if request.Body == nil {
		req, err = http.NewRequest(request.Route.Method, request.Route.Path, nil)
	} else {
		req, err = http.NewRequest(request.Route.Method, request.Route.Path, bytes.NewBuffer(request.Body))
	}

	if err != nil {
		return Response{}, err
	}

	for key, value := range request.Headers {
		req.Header.Add(key, value)
	}

	res := httptest.NewRecorder()
	router.ServeHTTP(res, req)

	p, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return Response{}, err
	}

	return Response{
		Code: res.Code,
		Body: p,
	}, nil
}
