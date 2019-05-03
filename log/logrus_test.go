package log

import (
	"bytes"
	"context"
	"io/ioutil"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestLoggerLevels(t *testing.T) {
	ctx := PutTraceID(context.Background(), int64(1234))
	fields := MapFields{"potato": "fried", "hamburger": "rare"}
	buffer := bytes.NewBufferString("")

	logger := NewLogrus(LogrusLoggerProperties{
		Level:     logrus.WarnLevel,
		Output:    buffer,
		Formatter: &logrus.JSONFormatter{TimestampFormat: "none"},
	})

	// debug logging should not log anything
	logger.Debug(ctx, "some message", fields)
	p, err := ioutil.ReadAll(buffer)

	assert.Nil(t, err)
	assert.Equal(t, "", string(p))

	// info logging should log normally the message
	logger.Info(ctx, "some message", fields)
	p, err = ioutil.ReadAll(buffer)

	assert.Nil(t, err)
	assert.Equal(t, "", string(p))

	// warn logging should log normally the message
	logger.Warn(ctx, "some message", fields)
	p, err = ioutil.ReadAll(buffer)

	assert.Nil(t, err)
	assert.Equal(t, "{"+
		"\"hamburger\":\"rare\","+
		"\"level\":\"warning\","+
		"\"msg\":\"some message\","+
		"\"potato\":\"fried\","+
		"\"time\":\"none\","+
		"\"traceId\":1234"+
		"}\n", string(p))

	// error logging should log normally the message
	logger.Error(ctx, "some message", fields)
	p, err = ioutil.ReadAll(buffer)

	assert.Nil(t, err)
	assert.Equal(t, "{"+
		"\"hamburger\":\"rare\","+
		"\"level\":\"error\","+
		"\"msg\":\"some message\","+
		"\"potato\":\"fried\","+
		"\"time\":\"none\","+
		"\"traceId\":1234"+
		"}\n", string(p))
}

func TestLoggerEntryLevels(t *testing.T) {
	ctx := PutTraceID(context.Background(), int64(1234))
	fields := MapFields{"potato": "fried", "hamburger": "rare"}
	buffer := bytes.NewBufferString("")

	logger := NewLogrus(LogrusLoggerProperties{
		Level:     logrus.DebugLevel,
		Output:    buffer,
		Formatter: &logrus.JSONFormatter{TimestampFormat: "none"},
	})

	entry := logger.ForClass("example", "MyStruct")

	// debug logging should not log anything
	entry.Debug(ctx, "some message", fields)
	p, err := ioutil.ReadAll(buffer)

	assert.Nil(t, err)
	assert.Equal(t, "{"+
		"\"class\":\"MyStruct\","+
		"\"hamburger\":\"rare\","+
		"\"level\":\"debug\","+
		"\"msg\":\"some message\","+
		"\"pkg\":\"example\","+
		"\"potato\":\"fried\","+
		"\"time\":\"none\","+
		"\"traceId\":1234"+
		"}\n", string(p))
}
