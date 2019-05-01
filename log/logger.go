package log

import "context"

type Fields interface {
	Add(key string, value interface{})
}

type Loggable interface {
	Log(fields Fields)
}

type MapFields map[string]interface{}

func (m MapFields) Log(fields Fields) {
	for key, value := range m {
		fields.Add(key, value)
	}
}

type Logger interface {
	ForClass(pkg string, class string) Logger
	Debug(ctx context.Context, msg string, loggable ...Loggable)
	Info(ctx context.Context, msg string, loggable ...Loggable)
	Warn(ctx context.Context, msg string, loggable ...Loggable)
	Error(ctx context.Context, msg string, loggable ...Loggable)
	Fatal(ctx context.Context, msg string, loggable ...Loggable)
}
