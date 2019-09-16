package core

import (
	"io/ioutil"
	"net/http"
	"testing"

	"github.com/oasislabs/developer-gateway/log"
	"github.com/oasislabs/developer-gateway/rpc"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

var Logger = log.NewLogrus(log.LogrusLoggerProperties{
	Level:  logrus.DebugLevel,
	Output: ioutil.Discard,
})

func TestServeHTTP(t *testing.T) {
	auth := &NilAuth{}
	handler := NewHttpMiddlewareAuth(auth, Logger, rpc.HttpMiddlewareFunc(func(req *http.Request) (interface{}, error) {
		assert.Equal(t, "nil", req.Context().Value(AAD{}))
		return 0, nil
	}))

	req, err := http.NewRequest("GET", "/", nil)
	assert.Nil(t, err)
	req.Header.Add(RequestHeaderSessionKey, "session")

	res, err := handler.ServeHTTP(req)
	assert.Nil(t, err)
	assert.Equal(t, 0, res)
}
