package http

import (
	"context"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
)

// ListenAndServeMonitoring starts up an HTTP server serving /metrics and /health.
//
// When the context is canceled, the server will be gracefully shutdown.
func ListenAndServeMonitoring(ctx context.Context, addr string, healthHandler http.Handler) error {
	srv := monitoringServer(addr, healthHandler)

	go func() {
		<-ctx.Done()
		shutdownContext, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()
		err := srv.Shutdown(shutdownContext)
		if err != nil {
			logrus.WithError(err).Error("failure during monitoring server shutdown")
		}
	}()

	return srv.ListenAndServe()
}

func monitoringServer(addr string, healthHandler http.Handler) *http.Server {
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	if healthHandler != nil {
		mux.Handle("/health", healthHandler)
	} else {
		mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
			_, err := w.Write([]byte(`{"msg":"ok"}`))
			if err != nil {
				logrus.WithError(err).Error("healthcheck write failure")
			}
		})
	}
	return &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}
}
