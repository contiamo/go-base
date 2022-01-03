package tracing

import (
	"net/http"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

// Middleware configures the otelhttp.NewHandler and returns a middleware is compatible
// with the chi router interface.
//
// Example:
//
// 	r := chi.NewRouter()
// 	r.Use(otel.Middleware("Server"))
//
// The most common options to pass will be the otelhttp.WithFilter and otelhttp.WithSpanNameFormatter
//
// When no options are passed a default filter for /metrics and /health is used.
func Middleware(name string, opts ...otelhttp.Option) func(next http.Handler) http.Handler {
	if name == "" {
		name = "Server"
	}
	if len(opts) == 0 {
		opts = []otelhttp.Option{WithInternalPathFilter}
	}

	return func(next http.Handler) http.Handler {
		return otelhttp.NewHandler(next, name, opts...)
	}
}

// WithInternalPathFilter is the default tracing filter passed to the Middleware.
// It filters out /metrics and /health requests.
var WithInternalPathFilter = otelhttp.WithFilter(func(r *http.Request) bool {
	switch r.URL.Path {
	case "/metrics", "/health":
		return true
	default:
		return false
	}
})
