package middlewares

import (
	"context"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	server "github.com/contiamo/go-base/v4/pkg/http"
)

func Test_MetricsMiddleware(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	t.Run("should be possible to configure metrics collection", func(t *testing.T) {
		srv, err := createServer([]server.Option{WithMetrics("test", nil)})
		require.NoError(t, err)

		ts := httptest.NewServer(srv.Handler)
		defer ts.Close()

		req, err := http.NewRequest(http.MethodGet, ts.URL+"/metrics_test", nil)
		require.NoError(t, err)
		req = req.WithContext(ctx)
		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		// it takes some time to run the server, can't be accessed immediately
		time.Sleep(200 * time.Millisecond)

		req, err = http.NewRequest(http.MethodGet, ts.URL+"/metrics", nil)
		require.NoError(t, err)
		req = req.WithContext(ctx)
		resp, err = http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		bs, _ := ioutil.ReadAll(resp.Body)

		countMetric := `http_request_total{code="200",instance="` + getHostname() + `",method="get",path="/metrics_test",service="test"} 1`
		durationMetric := `http_request_duration_ms_bucket{code="200",instance="` + getHostname() + `",method="get",path="/metrics_test",service="test",le="+Inf"} 1`
		sizeMetric := `http_response_size_bytes_bucket{code="200",instance="` + getHostname() + `",method="get",path="/metrics_test",service="test",le="+Inf"} 1`

		respBody := string(bs)
		require.Contains(t, respBody, countMetric)
		require.Contains(t, respBody, durationMetric)
		require.Contains(t, respBody, sizeMetric)
	})

	t.Run("should support websockets", func(t *testing.T) {
		srv, err := createServer([]server.Option{WithMetrics("test", nil)})
		require.NoError(t, err)
		ts := httptest.NewServer(srv.Handler)
		defer ts.Close()

		err = testWebsocketEcho(ts.URL)
		require.NoError(t, err)

		time.Sleep(100 * time.Millisecond)
		req, err := http.NewRequest(http.MethodGet, ts.URL+"/metrics", nil)
		require.NoError(t, err)
		req = req.WithContext(ctx)
		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		bs, _ := ioutil.ReadAll(resp.Body)

		countMetric := `http_request_duration_ms_bucket{code="0",instance="` + getHostname() + `",method="get",path="/ws/echo",service="test",le="+Inf"} 1`
		require.Contains(t, string(bs), countMetric)
	})
}
