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

	go ListenAndServeMonitoring(ctx, ":8080", nil)
	// it takes some time to run the server, can't be accessed immediately
	time.Sleep(100 * time.Millisecond)

	resp, err := http.Get("http://localhost:8080/metrics")
	require.NoError(t, err)

	defer resp.Body.Close()
	bs, _ := ioutil.ReadAll(resp.Body)
	require.Contains(t, string(bs), "go_info")
}
