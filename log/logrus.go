package log

import (
	"context"
	"io"
	"os"

	"github.com/sirupsen/logrus"
)

type LogrusLoggerProperties struct {
	Formatter logrus.Formatter
	Level     logrus.Level
	Output    io.Writer
}

type LogrusLogger struct {
	root *logrus.Logger
	log  *logrus.Logger
}

type LogrusEntry struct {
	root  *logrus.Logger
	entry *logrus.Entry
}

func NewLogrus(properties LogrusLoggerProperties) Logger {
	log := logrus.New()

	if properties.Formatter == nil {
		log.SetFormatter(&logrus.JSONFormatter{})
	} else {
		log.SetFormatter(properties.Formatter)
	}

	log.SetLevel(properties.Level)

	if properties.Output == nil {
		log.SetOutput(os.Stdout)
	} else {
		log.SetOutput(properties.Output)
	}

	return LogrusLogger{root: log, log: log}
}

func (l LogrusLogger) ForClass(pkg string, class string) Logger {
	return &LogrusEntry{
		root: l.root,
		entry: l.root.WithFields(logrus.Fields{
			"pkg":   pkg,
			"class": class,
		}),
	}
}

func (l LogrusLogger) Debug(ctx context.Context, msg string, loggables ...Loggable) {
	fields := logrusMakeFields(ctx, loggables...)
	l.log.WithFields(fields).Debug(msg)
}

func (l LogrusLogger) Info(ctx context.Context, msg string, loggables ...Loggable) {
	fields := logrusMakeFields(ctx, loggables...)
	l.log.WithFields(fields).Info(msg)
}

func (l LogrusLogger) Warn(ctx context.Context, msg string, loggables ...Loggable) {
	fields := logrusMakeFields(ctx, loggables...)
	l.log.WithFields(fields).Warn(msg)
}

func (l LogrusLogger) Error(ctx context.Context, msg string, loggables ...Loggable) {
	fields := logrusMakeFields(ctx, loggables...)
	l.log.WithFields(fields).Error(msg)
}

func (l LogrusLogger) Fatal(ctx context.Context, msg string, loggables ...Loggable) {
	fields := logrusMakeFields(ctx, loggables...)
	l.log.WithFields(fields).Fatal(msg)
}

func (e LogrusEntry) ForClass(pkg string, class string) Logger {
	return &LogrusEntry{
		root: e.root,
		entry: e.root.WithFields(logrus.Fields{
			"pkg":   pkg,
			"class": class,
		}),
	}
}

func (e LogrusEntry) Debug(ctx context.Context, msg string, loggables ...Loggable) {
	fields := logrusMakeFields(ctx, loggables...)
	e.entry.WithFields(fields).Debug(msg)
}

func (e LogrusEntry) Info(ctx context.Context, msg string, loggables ...Loggable) {
	fields := logrusMakeFields(ctx, loggables...)
	e.entry.WithFields(fields).Info(msg)
}

func (e LogrusEntry) Warn(ctx context.Context, msg string, loggables ...Loggable) {
	fields := logrusMakeFields(ctx, loggables...)
	e.entry.WithFields(fields).Warn(msg)
}

func (e LogrusEntry) Error(ctx context.Context, msg string, loggables ...Loggable) {
	fields := logrusMakeFields(ctx, loggables...)
	e.entry.WithFields(fields).Error(msg)
}

func (e LogrusEntry) Fatal(ctx context.Context, msg string, loggables ...Loggable) {
	fields := logrusMakeFields(ctx, loggables...)
	e.entry.WithFields(fields).Fatal(msg)
}

func logrusMakeFields(ctx context.Context, loggables ...Loggable) logrus.Fields {
	fields := LogrusFields{logrus.Fields{}}

	for _, loggable := range loggables {
		loggable.Log(&fields)
	}

	fields.Add("traceId", GetTraceID(ctx))
	return fields.fields
}

type LogrusFields struct {
	fields logrus.Fields
}

func (f *LogrusFields) Add(key string, value interface{}) {
	f.fields[key] = value
}
