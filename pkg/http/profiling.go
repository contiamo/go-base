package http

import (
	"context"
	"fmt"
	"net/http"
	"net/http/pprof"
	"time"

	"github.com/go-chi/chi"
)

// ListenAndServePprof starts a pprof server
func ListenAndServePprof(ctx context.Context, addr string) error {
	// setup handlers
	root := chi.NewRouter()
	root.Mount("/debug/pprof/", http.HandlerFunc(pprof.Index))
	root.Mount("/debug/pprof/cmdline", http.HandlerFunc(pprof.Cmdline))
	root.Mount("/debug/pprof/profile", http.HandlerFunc(pprof.Profile))
	root.Mount("/debug/pprof/symbol", http.HandlerFunc(pprof.Symbol))
	root.Mount("/debug/pprof/trace", http.HandlerFunc(pprof.Trace))
	srv := &http.Server{
		Addr:    addr,
		Handler: root,
	}

	// listen and serve profiling data
	done := make(chan error, 2) // 2 to prevent any blocking behavior even if 1 should be enough
	go func() { done <- srv.ListenAndServe() }()

	// wait for context to be canceled
	<-ctx.Done()

	// shutdown server
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	err := srv.Shutdown(shutdownCtx)
	if err != nil {
		done <- fmt.Errorf("failed to stop profiling server: %v", err)
	}

	// wait for termination of ListenAndServe or failed shutdown and return error
	return <-done
}
