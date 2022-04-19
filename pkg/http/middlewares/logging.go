package middlewares

import (
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/middleware"
	"github.com/sirupsen/logrus"
)

// WithLogging configures a logrus middleware for that server.
//
// You can control the log fields or if the log should be skipped by setting
// the Fields and the Ignore function respectively.
//
//      log := middlewares.WithLogging(config.ApplicationName)
//      log.Fields = func(r *http.Request) logrus.Fields {
//        // custom logic here
//      }
//      log.Ignore = func(r *http.Request) bool {
// 	      // custom logic here
//      }
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
func WithLogging(app string) *LoggingOption {
	return &LoggingOption{
		app: app,
	}
}

type LoggingOption struct {
	app string
	// Ignore determines if the log should be skipped for the given request.
	//
	// When nil, the default logic is used, which ignores requests in which the
	// User-Agent contains: "healthcheck" or "kube-probe".
	Ignore func(r *http.Request) bool
	// Fields extracts log fields from the given trequest to include in the log entry.
	//
	// When nil, the default logic is used, which includes the request method and path.
	Fields func(r *http.Request) logrus.Fields
}

func (opt *LoggingOption) WrapHandler(handler http.Handler) http.Handler {
	ignore := opt.Ignore
	if ignore == nil {
		ignore = defaultIgnore
	}

	fields := opt.Fields
	if fields == nil {
		fields = defaultFields
	}
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		resp := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
		handler.ServeHTTP(resp, r)
		// when the tracing middleware is initialized first _and_
		// the tracing.SpanHook is configured, then the logger will
		// also emit the trace and span ids as fields
		logger := logrus.
			WithContext(r.Context()).
			WithField("app", opt.app).
			WithFields(fields(r))

		duration := time.Since(start)
		status := resp.Status()
		if status == 0 {
			status = 200
		}
		logger = logger.WithFields(logrus.Fields{
			"duration_millis": duration.Nanoseconds() / 1000000,
			"status_code":     status,
		})
		if !isSuccess(status) {
			logger.Warn("problem while handling request")
			return
		}

		if !ignore(r) {
			logger.Info("successfully handled request")
		}
	})

	return h
}

func defaultFields(r *http.Request) logrus.Fields {
	return logrus.Fields{
		"method": r.Method,
		"path":   r.URL.EscapedPath(),
	}
}

func defaultIgnore(r *http.Request) bool {
	agent := strings.ToLower(r.Header.Get("User-Agent"))

	switch {
	case strings.Contains(agent, "healthcheck"), strings.Contains(agent, "kube-probe"):
		return true
	default:
		return false
	}
}

func isSuccess(status int) bool {
	return status >= 200 && status < 400
}
