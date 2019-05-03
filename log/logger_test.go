package log

import (
	"bytes"
	"context"
	"io/ioutil"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestMapFields(t *testing.T) {
	ctx := PutTraceID(context.Background(), int64(1234))
	fields := MapFields{"potato": "fried", "hamburger": "rare"}
	buffer := bytes.NewBufferString("")

	logger := NewLogrus(LogrusLoggerProperties{
		Level:     logrus.DebugLevel,
		Output:    buffer,
		Formatter: &logrus.JSONFormatter{TimestampFormat: "none"},
	})

	logger.Debug(ctx, "some message", fields)

	p, err := ioutil.ReadAll(buffer)

	assert.Nil(t, err)
	assert.Equal(t, "{"+
		"\"hamburger\":\"rare\","+
		"\"level\":\"debug\","+
		"\"msg\":\"some message\","+
		"\"potato\":\"fried\","+
		"\"time\":\"none\","+
		"\"traceId\":1234"+
		"}\n", string(p))
}
