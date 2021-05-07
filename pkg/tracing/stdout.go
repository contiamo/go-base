package tracing

import (
	"context"

	opentracing "github.com/opentracing/opentracing-go"
	otext "github.com/opentracing/opentracing-go/ext"
	otlog "github.com/opentracing/opentracing-go/log"
	"github.com/sirupsen/logrus"
	jaeger "github.com/uber/jaeger-client-go"
)

// NewTracerWithLogs create a new tracer that contains logs spans to stdout as well as to
// the opentracing tracer.
func NewTracerWithLogs(pkgName, componentName string) Tracer {
	return &loggingTracer{
		pkgName:       pkgName,
		componentName: componentName,
	}
}

type loggingTracer struct {
	pkgName, componentName string
}

// StartSpan implements BaseTracer with additional logging. A logrus logger is added to the context
func (t loggingTracer) StartSpan(ctx context.Context, operationName string) (opentracing.Span, context.Context) {
	span, ctx := opentracing.StartSpanFromContext(ctx, operationName)
	span.SetTag("pkg.name", t.pkgName)
	span.SetTag("pkg.component", t.componentName)

	traceID := GetTraceID(span)
	spanID := GetTraceID(span)

	// use underscore to be more compatible with Loki field parsing
	logrus.WithField("pkg_name", t.pkgName).
		WithField("pkg_component", t.componentName).
		WithField("operation", operationName).
		WithField("trace_id", traceID).
		WithField("span_id", spanID).
		Debug("start")

	return span, ctx
}

func (t loggingTracer) FinishSpan(span opentracing.Span, err error) {
	traceID := GetTraceID(span)
	spanID := GetTraceID(span)

	// use underscore to be more compatible with Loki field parsing
	// we also log the trace_id so that we can have grafana parse it
	// and link logs to the trace viewer
	logger := logrus.WithField("pkg_name", t.pkgName).
		WithField("pkg_component", t.componentName).
		WithField("trace_id", traceID).
		WithField("span_id", spanID)

	if err != nil {
		logger = logger.WithField("error", true)
		logger.Error(err.Error())
	}

	defer logger.Debug("finish")

	if span == nil {
		return
	}

	if err != nil {
		span.LogFields(
			otlog.String("error.msg", err.Error()),
		)
		otext.Error.Set(span, true)
	}
	span.Finish()
}

// GetTraceID extracts the span ID from the opentracing Span.
//
// Currently only jaeger is supported. This can be useful when paired
// with the latest Grafana that includes a trace viewer.
func GetTraceID(span opentracing.Span) string {
	switch s := span.(type) {
	case *jaeger.Span:
		return s.Context().(jaeger.SpanContext).TraceID().String()
	default:
		return ""
	}
}

// GetSpanID extracts the span ID from the opentracing Span.
//
// Currently only jaeger is supported. This can be useful when paired
// with the latest Grafana that includes a trace viewer.
func GetSpanID(span opentracing.Span) string {
	switch s := span.(type) {
	case *jaeger.Span:
		return s.Context().(jaeger.SpanContext).SpanID().String()
	default:
		return ""
	}
}
