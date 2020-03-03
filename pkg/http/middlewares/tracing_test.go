package middlewares

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	opentracing "github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/mocktracer"

	server "github.com/contiamo/go-base/v2/pkg/http"
)

func Test_TracingMiddleware(t *testing.T) {
	tracer := mocktracer.New()
	opentracing.SetGlobalTracer(tracer)

	t.Run("should be possible to setup tracing", func(t *testing.T) {
		srv, err := createServer([]server.Option{WithTracing("test", nil, nil)})
		require.NoError(t, err)

		ts := httptest.NewServer(srv.Handler)
		defer ts.Close()

		_, err = http.Get(ts.URL + "/tracing/")
		require.NoError(t, err)

		require.Len(t, tracer.FinishedSpans(), 1)
		tracer.Reset()
	})

	t.Run("should be possible to set additional tags", func(t *testing.T) {
		tagName := "testTag"
		tagValue := "something to find"
		srv, err := createServer([]server.Option{WithTracing("test", map[string]string{tagName: tagValue}, nil)})
		require.NoError(t, err)

		ts := httptest.NewServer(srv.Handler)
		defer ts.Close()

		_, err = http.Get(ts.URL + "/tracing/")
		require.NoError(t, err)

		require.Len(t, tracer.FinishedSpans(), 1)

		span := tracer.FinishedSpans()[0]
		require.Equal(t, tagValue, span.Tags()[tagName])
		tracer.Reset()
	})

	t.Run("should replace uuid values with *", func(t *testing.T) {
		srv, err := createServer([]server.Option{WithTracing("test", nil, nil)})
		require.NoError(t, err)

		ts := httptest.NewServer(srv.Handler)
		defer ts.Close()

		_, err = http.Get(ts.URL + "/tracing/2f6f97f2-5f44-476d-bc0c-180b2eaa36ca/2f6f97f2-5f44-476d-bc0c-180b2eaa36cb")
		require.NoError(t, err)

		require.Len(t, tracer.FinishedSpans(), 1)

		span := tracer.FinishedSpans()[0]
		require.Equal(t, "HTTP GET /tracing/*/*", span.OperationName)
		tracer.Reset()
	})

	t.Run("should allow websockets", func(t *testing.T) {
		srv, err := createServer([]server.Option{WithTracing("test", nil, nil)})
		require.NoError(t, err)

		ts := httptest.NewServer(srv.Handler)
		defer ts.Close()

		err = testWebsocketEcho(ts.URL)
		require.NoError(t, err)
	})
}
