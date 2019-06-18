package log

import (
	"github.com/sirupsen/logrus"
)

// New creates a new logger with the specified
// configuration
func New(config *Config) Logger {
	props := LogrusLoggerProperties{
		Level: logrus.DebugLevel,
	}

	switch config.Level {
	case "debug":
		props.Level = logrus.DebugLevel
	case "info":
		props.Level = logrus.InfoLevel
	case "warn":
		props.Level = logrus.WarnLevel
	default:
		props.Level = logrus.DebugLevel
	}

	return NewLogrus(props)
}
