package tracing

import (
	"context"

	opentracing "github.com/opentracing/opentracing-go"
)

// Finish creates a unified single finisher for spans, delegating the error
// handling to the existing TraceError method
func Finish(ctx context.Context, span opentracing.Span, err error) {
	if span == nil {
		return
	}

	TraceError(span, err)
	span.Finish()
}

// FinishGRPC creates a unified single finisher for spans inside a GRPC method.
// It delegating the error handling to the existing TraceGRPCError method
func FinishGRPC(ctx context.Context, span opentracing.Span, err error) {
	if span == nil {
		return
	}

	TraceGRPCError(span, err)
	span.Finish()
}
