package middlewares

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	server "github.com/contiamo/go-base/v4/pkg/http"
	utils "github.com/contiamo/go-base/v4/pkg/testing"
)

func Test_LoggingMiddleware(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	t.Run("should be possible to configure logging", func(t *testing.T) {
		buf, restore := utils.SetupLoggingBuffer()
		defer restore()

		srv, err := createServer([]server.Option{WithLogging("test")})
		require.NoError(t, err)

		ts := httptest.NewServer(srv.Handler)
		defer ts.Close()

		req, err := http.NewRequest(http.MethodGet, ts.URL+"/logging", nil)
		require.NoError(t, err)
		req = req.WithContext(ctx)

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		data := buf.String()
		require.Equal(t, 1, strings.Count(data, "\n"))
		require.Contains(t, data, `level=info`)
		require.Contains(t, data, `msg="successfully handled request"`)
		require.Contains(t, data, `app=test duration_millis=0 method=GET path=/logging status_code=200`)
	})

	t.Run("should support websockets", func(t *testing.T) {
		buf, restore := utils.SetupLoggingBuffer()
		defer restore()

		srv, err := createServer([]server.Option{WithLogging("test")})
		require.NoError(t, err)

		ts := httptest.NewServer(srv.Handler)
		defer ts.Close()

		err = testWebsocketEcho(ts.URL)
		require.NoError(t, err)

		time.Sleep(100 * time.Millisecond)
		data := buf.String()
		require.Equal(t, 1, strings.Count(data, "\n"))
		require.Contains(t, data, `level=info`)
		require.Contains(t, data, `msg="successfully handled request"`)
		require.Contains(t, data, `app=test duration_millis=0 method=GET path=/ws/echo status_code=200`)
	})

	t.Run("should ignore healthchecks", func(t *testing.T) {
		buf, restore := utils.SetupLoggingBuffer()
		defer restore()

		srv, err := createServer([]server.Option{WithLogging("test")})
		require.NoError(t, err)

		ts := httptest.NewServer(srv.Handler)
		defer ts.Close()

		req, err := http.NewRequest(http.MethodGet, ts.URL+"/logging", nil)
		require.NoError(t, err)
		req = req.WithContext(ctx)
		req.Header.Set("User-Agent", "ELB-HealthChecker")

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		require.Equal(t, 200, resp.StatusCode)

		require.Equal(t, buf.String(), "")
	})

	t.Run("should log on error codes", func(t *testing.T) {
		buf, restore := utils.SetupLoggingBuffer()
		defer restore()

		srv, err := createServer([]server.Option{WithLogging("test")})
		require.NoError(t, err)

		ts := httptest.NewServer(srv.Handler)
		defer ts.Close()

		req, err := http.NewRequest(http.MethodGet, ts.URL+"/error", nil)
		require.NoError(t, err)
		req = req.WithContext(ctx)

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		require.Equal(t, 400, resp.StatusCode)

		data := buf.String()
		require.Equal(t, 1, strings.Count(data, "\n"), data)
		require.Contains(t, data, `level=warn`)
		require.Contains(t, data, `msg="problem while handling request"`)
		require.Contains(t, data, `app=test duration_millis=0 method=GET path=/error status_code=400`)
	})
}
