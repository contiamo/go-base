package tracing

import (
	"errors"
	"io/ioutil"
	"os"
	"testing"

	"github.com/opentracing/opentracing-go"

	"google.golang.org/grpc/status"

	"google.golang.org/grpc/codes"

	"github.com/opentracing/opentracing-go/mocktracer"
	"github.com/sirupsen/logrus"
	ltest "github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/require"
)

func Test_TraceError(t *testing.T) {
	logrus.SetLevel(logrus.DebugLevel)
	logrus.SetOutput(ioutil.Discard)

	hook := ltest.NewGlobal()
	tracer := mocktracer.New()

	cases := []struct {
		name string
		span opentracing.Span
		err  error
		tags map[string]interface{}
	}{
		{"non-nil error tags the span and causes Debug log", tracer.StartSpan("TestSpan"), errors.New("Test Error Message"), map[string]interface{}{"error": true}},
		{"nil error does not tag the span or cause a log", tracer.StartSpan("TestSpan"), nil, map[string]interface{}{}},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			TraceError(c.span, c.err)
			mockSpan := c.span.(*mocktracer.MockSpan)

			require.Equal(t, c.tags, mockSpan.Tags())
			if c.err != nil {
				require.Equal(t, 1, len(hook.Entries))
				require.Equal(t, logrus.DebugLevel, hook.LastEntry().Level)
				require.Equal(t, c.err.Error(), hook.LastEntry().Message)
			}

			hook.Reset()
		})
	}

	logrus.SetOutput(os.Stdout)
}

func Test_TraceGRPCError(t *testing.T) {
	logrus.SetLevel(logrus.DebugLevel)
	logrus.SetOutput(ioutil.Discard)
	hook := ltest.NewGlobal()
	tracer := mocktracer.New()

	cases := []struct {
		name string
		span opentracing.Span
		err  error
		tags map[string]interface{}
	}{
		{"non-nil error tags the span and causes Debug log", tracer.StartSpan("TestSpan"), errors.New("Test Error Message"), map[string]interface{}{"error": true, "error.code": codes.Unknown.String()}},
		{"non-nil error tags the span and causes Debug log", tracer.StartSpan("TestSpan"), status.Error(codes.InvalidArgument, "Test Error Message"), map[string]interface{}{"error": true, "error.code": codes.InvalidArgument.String()}},
		{"nil error does not tag the span or cause a log", tracer.StartSpan("TestSpan"), nil, map[string]interface{}{"error.code": codes.OK.String()}},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			TraceGRPCError(c.span, c.err)
			mockSpan := c.span.(*mocktracer.MockSpan)

			require.Equal(t, c.tags, mockSpan.Tags())
			if c.err != nil {
				require.Equal(t, 1, len(hook.Entries))
				require.Equal(t, logrus.DebugLevel, hook.LastEntry().Level)
				require.Equal(t, c.err.Error(), hook.LastEntry().Message)
			}

			hook.Reset()
		})
	}
	logrus.SetOutput(os.Stdout)
}
