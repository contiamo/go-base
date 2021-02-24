package middlewares

import (
	"net/http"
	"net/http/httptest"
	"testing"

	server "github.com/contiamo/go-base/v3/pkg/http"
	"github.com/stretchr/testify/require"
)

func Test_CORSMiddleware(t *testing.T) {
	t.Run("should be possible to configure custom CORS rules", func(t *testing.T) {
		allowedOrigins := []string{"foo.bar"}
		allowedMethods := []string{"HEAD"}
		allowedHeaders := []string{"Content-Type"}
		allowCredentials := true
		srv, err := createServer([]server.Option{WithCORS(allowedOrigins, allowedMethods, allowedHeaders, allowCredentials)})
		require.NoError(t, err)

		ts := httptest.NewServer(srv.Handler)
		defer ts.Close()

		req, _ := http.NewRequest(http.MethodOptions, ts.URL+"/cors", nil)
		req.Header.Set("Access-Control-Request-Method", "HEAD")
		req.Header.Set("Origin", "foo.bar")

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)

		require.Equal(t, "true", resp.Header.Get("Access-Control-Allow-Credentials"))
		require.Equal(t, "foo.bar", resp.Header.Get("Access-Control-Allow-Origin"))
		require.Equal(t, "HEAD", resp.Header.Get("Access-Control-Allow-Methods"))
	})

	t.Run("should support websockets", func(t *testing.T) {
		srv, err := createServer([]server.Option{WithCORSWideOpen()})
		require.NoError(t, err)

		ts := httptest.NewServer(srv.Handler)
		defer ts.Close()

		err = testWebsocketEcho(ts.URL)
		require.NoError(t, err)
	})
}
