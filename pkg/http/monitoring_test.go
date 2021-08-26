package http

import (
	"context"
	"io/ioutil"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestListenAndServeMonitoring(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	go func() {
		_ = ListenAndServeMonitoring(ctx, ":8080", nil)
	}()
	// it takes some time to run the server, can't be accessed immediately
	time.Sleep(100 * time.Millisecond)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://localhost:8080/metrics", nil)
	require.NoError(t, err)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)

	defer resp.Body.Close()
	bs, err := ioutil.ReadAll(resp.Body)
	require.NoError(t, err)
	require.Contains(t, string(bs), "go_info")
}

func TestListenAndServeMonitoringShutdown(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)

	done := make(chan error)
	go func() {
		err := ListenAndServeMonitoring(ctx, ":8080", nil)
		done <- err
	}()
	// it takes some time to run the server, can't be accessed immediately
	time.Sleep(100 * time.Millisecond)
	cancel()

	err := <-done
	require.EqualError(t, err, http.ErrServerClosed.Error())
}
