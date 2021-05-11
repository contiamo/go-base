package middlewares

import (
	"net/http"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/urfave/negroni"

	server "github.com/contiamo/go-base/v3/pkg/http"
)

// WithLogging configures a logrus middleware for that server
func WithLogging(app string) server.Option {
	return &loggingOption{app}
}

type loggingOption struct{ app string }

func (opt *loggingOption) WrapHandler(handler http.Handler) http.Handler {
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		handler.ServeHTTP(w, r)
		logger := logrus.WithFields(logrus.Fields{
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
		logger = logrus.WithFields(logrus.Fields{
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
