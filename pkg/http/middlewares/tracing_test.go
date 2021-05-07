package middlewares

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	opentracing "github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/mocktracer"

	server "github.com/contiamo/go-base/v3/pkg/http"
)

func Test_TracingMiddleware(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	tracer := mocktracer.New()
	opentracing.SetGlobalTracer(tracer)

	t.Run("should be possible to setup tracing", func(t *testing.T) {
		srv, err := createServer([]server.Option{WithTracing("test", nil, nil)})
		require.NoError(t, err)

		ts := httptest.NewServer(srv.Handler)
		defer ts.Close()

		req, err := http.NewRequest(http.MethodGet, ts.URL+"/tracing", nil)
		require.NoError(t, err)
		req = req.WithContext(ctx)
		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

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

		req, err := http.NewRequest(http.MethodGet, ts.URL+"/tracing", nil)
		require.NoError(t, err)
		req = req.WithContext(ctx)
		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

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

		req, err := http.NewRequest(http.MethodGet, ts.URL+"/tracing/2f6f97f2-5f44-476d-bc0c-180b2eaa36ca/2f6f97f2-5f44-476d-bc0c-180b2eaa36cb", nil)
		require.NoError(t, err)
		req = req.WithContext(ctx)
		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

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
