package http

import (
	"context"
	"net/http"
	"time"

	"github.com/sirupsen/logrus"
)

// Config contains the http servers config options
type Config struct {
	Addr    string
	Handler http.Handler
	Options []Option
}

// New creates a new http server
//
// Example:
//
// srv, _ := server.New(&server.Config{
//  Addr: ":8080",
//  Handler: http.DefaultServeMux,
//  Options: []server.Option{
//    server.WithLogging("my-server"),
//    server.WithMetrics("my-server"),
//    server.WithRecovery(),
//    server.WithTracing("opentracing-server:6831", "my-server"),
//  }
// })
//
// srv.ListenAndServe(context.Background())
func New(cfg *Config) (*http.Server, error) {
	var (
		h = cfg.Handler
	)
	for _, opt := range cfg.Options {
		h = opt.WrapHandler(h)
	}
	return &http.Server{
		Handler:        h,
		Addr:           cfg.Addr,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}, nil
}

// ListenAndServe serves an http server over TCP
func ListenAndServe(ctx context.Context, addr string, srv *http.Server) error {
	if addr != "" {
		srv.Addr = addr
	}
	go func() {
		<-ctx.Done()
		shutdownContext, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()
		srv.Shutdown(shutdownContext)
	}()
	logrus.Info("start listening for HTTP requests on " + srv.Addr)
	return srv.ListenAndServe()
}

// Option is the interface for all server options defined in this package
type Option interface {
	WrapHandler(handler http.Handler) http.Handler
}
