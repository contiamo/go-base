package middlewares

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	server "github.com/contiamo/go-base/v4/pkg/http"
	"github.com/stretchr/testify/require"
)

func Test_CORSMiddleware(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

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
		req = req.WithContext(ctx)
		req.Header.Set("Access-Control-Request-Method", "HEAD")
		req.Header.Set("Origin", "foo.bar")

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

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
