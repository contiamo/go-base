package middlewares

import (
	"net/http"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/urfave/negroni"

	server "github.com/contiamo/go-base/v3/pkg/http"
)

// WithLogging configures a logrus middleware for that server.
//
// When `tracing.SpanHook` is enabled and the tracing middleware is enabled
// before the logging middleware, the traceId and spanId are attaced the the logs.
//
// During application configuration use
//
// 		logrus.AddHook(&tracing.SpanHook{})
//
//
// During router configuration
//
// 		recover := middlewares.WithRecovery(os.Stderr, cfg.Debug)
// 		trace := middlewares.WithTracing(config.ApplicationName, nil, middlewares.ChiRouteName)
// 		log := middlewares.WithLogging(config.ApplicationName)
// 		metrics := middlewares.WithMetrics(config.ApplicationName, nil)
//
// 		api.Use(
// 			recover.WrapHandler,
// 			trace.WrapHandler,
// 			log.WrapHandler,
// 			metrics.WrapHandler,
// 		)
func WithLogging(app string) server.Option {
	return &loggingOption{app}
}

type loggingOption struct{ app string }

func (opt *loggingOption) WrapHandler(handler http.Handler) http.Handler {
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		handler.ServeHTTP(w, r)
		// when the tracing middleware is initialized first _and_
		// the tracing.SpanHook is configured, then the logger will
		// also emit the trace and span ids as fields
		logger := logrus.
			WithContext(r.Context()).
			WithFields(logrus.Fields{
				"app":  opt.app,
				"path": r.URL.EscapedPath(),
			})
		resp, ok := w.(negroni.ResponseWriter)
		if !ok {
			logger.Warn("wrong request type")
			return
		}
		duration := time.Since(start)
		status := resp.Status()
		if status == 0 {
			status = 200
		}
		logger = logger.WithFields(logrus.Fields{
			"duration_millis": duration.Nanoseconds() / 1000000,
			"status_code":     status,
		})
		if status >= 200 && status < 400 {
			logger.Info("successfully handled request")
		} else {
			logger.Warn("problem while handling request")
		}
	})
	n := negroni.New()
	n.UseHandler(h)
	return n
}
