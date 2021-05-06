package middlewares

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	server "github.com/contiamo/go-base/v3/pkg/http"
	utils "github.com/contiamo/go-base/v3/pkg/testing"
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

		require.Contains(t, buf.String(), "successfully handled request")
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
		require.Contains(t, buf.String(), "successfully handled request")
	})
}
