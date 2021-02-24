package middlewares

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	server "github.com/contiamo/go-base/v3/pkg/http"
	utils "github.com/contiamo/go-base/v3/pkg/testing"
)

func Test_LoggingMiddleware(t *testing.T) {

	t.Run("should be possible to configure logging", func(t *testing.T) {
		buf, restore := utils.SetupLoggingBuffer()
		defer restore()

		srv, err := createServer([]server.Option{WithLogging("test")})
		require.NoError(t, err)

		ts := httptest.NewServer(srv.Handler)
		defer ts.Close()

		_, err = http.Get(ts.URL + "/logging")
		require.NoError(t, err)

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
