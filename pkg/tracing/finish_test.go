package tracing

import (
	"context"
	"io/ioutil"
	"testing"

	"github.com/stretchr/testify/require"

	opentracing "github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/mocktracer"
	"github.com/sirupsen/logrus"
)

func Test_Finish(t *testing.T) {
	logrus.SetOutput(ioutil.Discard)
	tracer := mocktracer.New()
	ctx := context.TODO()

	cases := []struct {
		name string
		span opentracing.Span
	}{
		{"non-nil span is finished", tracer.StartSpan("TestSpan")},
		{"nil span is a no-op", nil},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			Finish(ctx, c.span, nil)
			if c.span != nil {
				mockSpan := c.span.(*mocktracer.MockSpan)
				require.NotEmpty(t, mockSpan.FinishTime)
			}
		})
	}
}

func Test_FinishGRPC(t *testing.T) {
	logrus.SetOutput(ioutil.Discard)
	tracer := mocktracer.New()
	ctx := context.TODO()

	cases := []struct {
		name string
		span opentracing.Span
	}{
		{"non-nil span is finished", tracer.StartSpan("TestSpan")},
		{"nil span is a no-op", nil},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			FinishGRPC(ctx, c.span, nil)
			if c.span != nil {
				mockSpan := c.span.(*mocktracer.MockSpan)
				require.NotEmpty(t, mockSpan.FinishTime)
			}
		})
	}
}
