package middlewares

import (
	"net/http"

	server "github.com/contiamo/go-base/v2/pkg/http"

	"github.com/rs/cors"
)

// WithCORS configures CORS on the webserver
func WithCORS(allowedOrigins, allowedMethods, allowedHeaders []string, allowCredentials bool) server.Option {
	return &corsOption{allowedOrigins, allowedMethods, allowedHeaders, allowCredentials}
}

// WithCORSWideOpen allows requests from all origins with all methods and all headers/cookies/credentials allowed.
func WithCORSWideOpen() server.Option {
	return &corsOption{
		allowedOrigins:   []string{"*"},
		allowedMethods:   []string{"HEAD", "GET", "POST", "PUT", "PATCH", "DELETE"},
		allowedHeaders:   []string{"*"},
		allowCredentials: true,
	}
}

type corsOption struct {
	allowedOrigins   []string
	allowedMethods   []string
	allowedHeaders   []string
	allowCredentials bool
}

func (opt *corsOption) WrapHandler(handler http.Handler) http.Handler {
	c := cors.New(cors.Options{
		AllowedOrigins:   opt.allowedOrigins,
		AllowedMethods:   opt.allowedMethods,
		AllowedHeaders:   opt.allowedHeaders,
		AllowCredentials: opt.allowCredentials,
	})
	return c.Handler(handler)
}
