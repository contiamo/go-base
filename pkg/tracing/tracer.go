package tracing

import (
	"context"

	opentracing "github.com/opentracing/opentracing-go"
	otext "github.com/opentracing/opentracing-go/ext"
	otlog "github.com/opentracing/opentracing-go/log"
	"github.com/sirupsen/logrus"
)

// Tracer contains all the tracing-related functions
// for any module of the server that uses tracing
type Tracer interface {
	StartSpan(ctx context.Context, operationName string) (opentracing.Span, context.Context)
	FinishSpan(opentracing.Span, error)
}

// NewTracer create a new tracer that contains implementation for all tracing-related actions
func NewTracer(pkgName, componentName string) Tracer {
	return &tracer{
		pkgName:       pkgName,
		componentName: componentName,
	}
}

type tracer struct {
	pkgName, componentName string
}

func (t tracer) StartSpan(ctx context.Context, operationName string) (opentracing.Span, context.Context) {
	span, ctx := opentracing.StartSpanFromContext(ctx, operationName)
	span.SetTag("pkg.name", t.pkgName)
	span.SetTag("pkg.component", t.componentName)
	return span, ctx
}

func (t tracer) FinishSpan(span opentracing.Span, err error) {
	if err != nil {
		logrus.WithField("package", t.pkgName).WithField("component", t.componentName).Error(err.Error())
	}

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
