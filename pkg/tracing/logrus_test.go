package tracing

import (
	"context"
	"fmt"
	"testing"

	"github.com/opentracing/opentracing-go/mocktracer"

	"github.com/opentracing/opentracing-go"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

func TestSpanHook(t *testing.T) {

	opentracing.SetGlobalTracer(mocktracer.New())

	hook := &SpanHook{}
	ctx := context.Background()
	span, ctxWithSpan := opentracing.StartSpanFromContext(ctx, "TestSpanHook")

	mockspan, ok := span.(*mocktracer.MockSpan)
	require.True(t, ok, "must have mock span for a valid test case")


	fields := logrus.Fields{
		"count": 2,
		"welcome": "hi there",
	}

	cases := []struct{
		name string
		entry *logrus.Entry
		span *mocktracer.MockSpan
		err string
	}{
		{
			name: "handles missing context",
			entry: &logrus.Entry{
				Message: "test message without context",
			},
		},
		{
			name: "handles context with no span",
			entry: &logrus.Entry{
				Message: "test message without span",
				Context: context.Background(),
			},
		},
		{
			name: "handles context with valid span",
			entry: &logrus.Entry{
				Message: "test message with tracing",
				Context: ctxWithSpan,
				Data: fields,
			},
			span: mockspan,
		},
	}

	for _, tc := range cases{
		t.Run(tc.name, func(t *testing.T) {
			err := hook.Fire(tc.entry)
			if tc.err != "" {
				require.EqualError(t, err, tc.err)
				return
			}

			require.NoError(t, err)
			if tc.span != nil {
				spanFields := map[string]string{}
				logs := tc.span.Logs()
				for _, record := range logs {
					for _, f := range record.Fields {
						spanFields[f.Key] = f.ValueString
					}
				}

				entryFields := map[string]string{
					"log.msg": tc.entry.Message,
				}
				for name, value := range tc.entry.Data {
					entryFields[name] = fmt.Sprintf("%v", value)
				}

				require.Equal(t, entryFields, spanFields, "log fields mismatch")
			}

		})
	}
}
