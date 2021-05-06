package tracing

import (
	"fmt"

	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/log"
	"github.com/opentracing/opentracing-go/mocktracer"
	"github.com/sirupsen/logrus"
	jaeger "github.com/uber/jaeger-client-go"
)

// SpanHook is a logrus Hook to send write logs and their fields to the current span.
type SpanHook struct{}

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

	switch sc := span.Context().(type) {
	case jaeger.SpanContext:
		entry.Data["traceId"] = sc.TraceID().String()
		entry.Data["spanId"] = sc.SpanID().String()
	case mocktracer.MockSpanContext:
		entry.Data["traceId"] = fmt.Sprintf("%v", sc.TraceID)
		entry.Data["spanId"] = fmt.Sprintf("%v", sc.SpanID)
	}
	return nil
}

// Levels implements the Hook interface
func (hook *SpanHook) Levels() []logrus.Level {
	return logrus.AllLevels
}
