package apitest

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"

	auth "github.com/oasislabs/developer-gateway/auth/core"
	"github.com/oasislabs/developer-gateway/auth/insecure"
	"github.com/oasislabs/developer-gateway/rpc"
)

// NewClient creates a new client for calling the API
func NewClient(router *rpc.HttpRouter) *Client {
	return &Client{
		router: router,
	}
}

// Client is an implementation of a client that sends requests
// to a bound server. The server is just the API implementation
// this is not supposed to be a client to make requests with
// real IO, but just to test a particular implementation
type Client struct {
	// router is the server implementation that the client
	// sends requests to
	router *rpc.HttpRouter
}

// Request performs a request and returns the response generated
// by the server, or the error if any
func (c *Client) Request(req Request) (Response, error) {
	httpRequest, err := c.createHTTPRequest(req)
	if err != nil {
		return Response{}, err
	}

	for key, value := range req.Headers {
		httpRequest.Header.Add(key, value)
	}

	httpResponse := httptest.NewRecorder()
	c.router.ServeHTTP(httpResponse, httpRequest)
	p, err := ioutil.ReadAll(httpResponse.Body)
	if err != nil {
		return Response{}, err
	}

	return Response{
		Code: httpResponse.Code,
		Body: p,
	}, nil
}

// RequestAPI performs a request to the API
func (c *Client) RequestAPI(de rpc.Deserializer, req interface{}, session string, route Route) error {
	p, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal request body %s", err.Error())
	}

	res, err := c.Request(Request{
		Route: route,
		Body:  p,
		Headers: map[string]string{
			insecure.HeaderKey:           "mykey",
			auth.RequestHeaderSessionKey: session,
			"Content-type":               "application/json",
		},
	})
	if err != nil {
		return err
	}

	if res.Code != http.StatusOK {
		if len(res.Body) > 0 {
			var rpcError rpc.Error
			if err := json.Unmarshal(res.Body, &rpcError); err != nil {
				fmt.Errorf("failed to unmarshal response body as error %s", err.Error())
			}
			return &rpcError
		}

		return &rpc.HttpError{Cause: nil, StatusCode: res.Code}
	}

	if err := de.Deserialize(bytes.NewBuffer(res.Body)); err != nil {
		return fmt.Errorf("failed to unmarshal response body %s", err.Error())
	}

	return nil
}

func (c *Client) createHTTPRequest(req Request) (*http.Request, error) {
	if req.Body == nil {
		return http.NewRequest(req.Route.Method, req.Route.Path, nil)
	} else {
		return http.NewRequest(req.Route.Method, req.Route.Path, bytes.NewBuffer(req.Body))
	}
}
