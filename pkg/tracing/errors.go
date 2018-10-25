package tracing

import (
	opentracing "github.com/opentracing/opentracing-go"
	otext "github.com/opentracing/opentracing-go/ext"
	otlog "github.com/opentracing/opentracing-go/log"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/status"
)

// TraceError sets the error flag and the error message on the span, this is a noop if the
// err or the span is nil
func TraceError(span opentracing.Span, err error) {
	if err == nil || span == nil {
		return
	}
	otext.Error.Set(span, true)
	span.LogFields(
		otlog.String("error.msg", err.Error()),
	)
	logrus.Debug(err.Error())
}

// TraceGRPCError calls TraceError setting additional GRPC-related tags to the span.
func TraceGRPCError(span opentracing.Span, err error) {
	span.SetTag("error.code", status.Code(err).String())
	TraceError(span, err)
}
