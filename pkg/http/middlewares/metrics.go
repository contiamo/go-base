package middleware

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/urfave/negroni"

	server "github.com/contiamo/go-base/pkg/http"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	durationMsBuckets = []float64{10, 50, 100, 200, 300, 500, 1000, 2000, 3000, 5000, 10000, 15000, 20000, 30000}
	sizeBytesBuckets  = []float64{1, 2, 4, 8, 16, 32, 64, 128, 256, 512, 1024, 2048, 4096, 8192, 16384, 32768, 65536, 131072, 262144, 524288, 1048576, 2097152, 4194304}
)

// WithMetrics configures metrics collection
func WithMetrics(app string, opNameFunc func(r *http.Request) string) server.Option {
	if opNameFunc == nil {
		opNameFunc = PathWithCleanID
	}

	constLabels := prometheus.Labels{"service": app, "instance": getHostname()}

	requestDuration := prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace:   "http",
		Subsystem:   "request",
		Name:        "duration_ms",
		Help:        "The duration of a request in milliseconds by status, method, and path.",
		ConstLabels: constLabels,
		Buckets:     durationMsBuckets,
	},
		[]string{"code", "method", "path"},
	)
	requestCounter := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace:   "http",
			Subsystem:   "request",
			Name:        "total",
			Help:        "Count of the requests by status, method, and path.",
			ConstLabels: constLabels,
		},
		[]string{"code", "method", "path"},
	)
	responseSize := prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace:   "http",
		Subsystem:   "response",
		Name:        "size_bytes",
		Help:        "The size of the response in bytes by status, method, and path.",
		ConstLabels: constLabels,
		Buckets:     sizeBytesBuckets,
	},
		[]string{"code", "method", "path"},
	)
	prometheus.Unregister(requestDuration)
	prometheus.Unregister(requestCounter)
	prometheus.Unregister(responseSize)

	prometheus.MustRegister(requestDuration, requestCounter, responseSize)

	return &metricsOption{app, opNameFunc, requestDuration, requestCounter, responseSize}
}

type metricsOption struct {
	app             string
	opNameFunc      func(r *http.Request) string
	requestDuration *prometheus.HistogramVec
	requestCounter  *prometheus.CounterVec
	responseSize    *prometheus.HistogramVec
}

func (opt *metricsOption) WrapHandler(handler http.Handler) http.Handler {

	mw := http.HandlerFunc(func(writer http.ResponseWriter, r *http.Request) {
		instrumentedWriter := negroni.NewResponseWriter(writer)

		defer func(begun time.Time) {
			l := prometheus.Labels{
				"code":   strconv.Itoa(instrumentedWriter.Status()),
				"method": strings.ToLower(r.Method),
				"path":   opt.opNameFunc(r),
			}

			opt.requestCounter.With(l).Inc()
			opt.requestDuration.With(l).Observe(float64(time.Since(begun).Seconds() * 1000))
			opt.responseSize.With(l).Observe(float64(instrumentedWriter.Size()))
		}(time.Now())

		handler.ServeHTTP(instrumentedWriter, r)
	})

	return mw
}
