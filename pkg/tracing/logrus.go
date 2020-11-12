package tracing

import (
	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/log"
	"github.com/sirupsen/logrus"
)

// SpanHook is a logrus Hook to send write logs and their fields to the current span.
type SpanHook struct {}

// Fire implements the Hook interface
func (hook *SpanHook) Fire(entry *logrus.Entry) error {
	if entry == nil {
		return nil
	}

	ctx := entry.Context
	if ctx == nil {
		return nil
	}

	span := opentracing.SpanFromContext(ctx)
	if span == nil {
		return nil
	}

	fields := []log.Field{
		log.String("log.msg", entry.Message),
	}

	for name, data := range entry.Data {
		fields = append(fields, log.Object(name, data))
	}

	span.LogFields(fields...)


	return nil
}

// Levels implements the Hook interface
func (hook *SpanHook) Levels() []logrus.Level {
	return logrus.AllLevels
}
