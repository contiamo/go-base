package middlewares

import (
	"context"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	server "github.com/contiamo/go-base/v3/pkg/http"
)

func Test_RecoveryMiddleware(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	t.Run("should be possible to configure panic recovery", func(t *testing.T) {
		srv, err := createServer([]server.Option{WithRecovery(ioutil.Discard, true)})
		require.NoError(t, err)

		ts := httptest.NewServer(srv.Handler)
		defer ts.Close()

		req, err := http.NewRequest(http.MethodGet, ts.URL+"/panic", nil)
		require.NoError(t, err)
		req = req.WithContext(ctx)
		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		require.Equal(t, http.StatusInternalServerError, resp.StatusCode)
	})

	t.Run("should support websockets and tracing", func(t *testing.T) {
		srv, err := createServer([]server.Option{WithRecovery(ioutil.Discard, true)})
		require.NoError(t, err)

		ts := httptest.NewServer(srv.Handler)
		defer ts.Close()

		err = testWebsocketEcho(ts.URL)
		require.NoError(t, err)
	})
}
