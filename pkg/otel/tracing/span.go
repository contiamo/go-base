package tracing

import (
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// RecordError is a utility to simplify recording an error to the
// OpenTelemetry span and setting the error status code.
func RecordError(span trace.Span, err error) {
	if err == nil {
		return
	}

	if span == nil {
		return
	}

	span.RecordError(err)
	span.SetStatus(codes.Error, err.Error())
}

// EndSpan is a utility to simplify ending a span and recording an error.
func EndSpan(span trace.Span, err error) {
	if err == nil {
		span.End()
		return
	}

	span.RecordError(err)
	span.SetStatus(codes.Error, err.Error())
	span.End()
}
