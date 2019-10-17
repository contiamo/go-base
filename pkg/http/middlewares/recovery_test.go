package middlewares

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	server "github.com/contiamo/go-base/pkg/http"
)

func Test_RecoveryMiddleware(t *testing.T) {
	t.Run("should be possible to configure panic recovery", func(t *testing.T) {
		srv, err := createServer([]server.Option{WithRecovery(ioutil.Discard, true)})
		require.NoError(t, err)

		ts := httptest.NewServer(srv.Handler)
		defer ts.Close()

		resp, err := http.Get(ts.URL + "/panic")
		require.NoError(t, err)
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
