package middlewares

import (
	"net/http"

	opentracing "github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	"github.com/urfave/negroni"

	server "github.com/contiamo/go-base/v4/pkg/http"
	"github.com/contiamo/go-base/v4/pkg/tracing"
)

// mainly from "github.com/opentracing-contrib/go-stdlib/nethttp"

// WithTracing configures tracing for that server; if configuration fails, WithTracing will panic
func WithTracing(app string, tags map[string]string, opNameFunc func(r *http.Request) string) server.Option {
	if opNameFunc == nil {
		opNameFunc = MethodAndPathCleanID
	}
	if err := tracing.InitJaeger(app); err != nil {
		panic(err)
	}
	return &tracingOption{app, tags, opNameFunc}
}

type tracingOption struct {
	app        string
	tags       map[string]string
	opNameFunc func(r *http.Request) string
}

func (opt *tracingOption) WrapHandler(handler http.Handler) http.Handler {
	mw := traceMiddleware(
		opentracing.GlobalTracer(),
		handler,
		operationNameFunc(opt.opNameFunc),
		mwSpanObserver(func(sp opentracing.Span, r *http.Request) {
			sp.SetTag("http.uri", r.URL.EscapedPath())
			for k, v := range opt.tags {
				sp.SetTag(k, v)
			}
		}),
		mwComponentName(opt.app),
	)
	n := negroni.New()
	n.UseHandler(mw)
	return n
}

type mwOptions struct {
	opNameFunc    func(r *http.Request) string
	spanObserver  func(span opentracing.Span, r *http.Request)
	componentName string
}

// mwOption controls the behavior of the Middleware.
type mwOption func(*mwOptions)

// OperationNameFunc returns a mwOption that uses given function f
// to generate operation name for each server-side span.
func operationNameFunc(f func(r *http.Request) string) mwOption {
	return func(options *mwOptions) {
		options.opNameFunc = f
	}
}

// mwSpanObserver returns a MWOption that observe the span
// for the server-side span.
func mwSpanObserver(f func(span opentracing.Span, r *http.Request)) mwOption {
	return func(options *mwOptions) {
		options.spanObserver = f
	}
}

// MWComponentName returns a mwOption that sets the component name
// for the server-side span.
func mwComponentName(componentName string) mwOption {
	return func(options *mwOptions) {
		options.componentName = componentName
	}
}

// Middleware wraps an http.Handler and traces incoming requests.
// Additionally, it adds the span to the request's context.
//
// By default, the operation name of the spans is set to "HTTP {method}".
// This can be overridden with options.
//
// Example:
//
//	http.ListenAndServe("localhost:80", nethttp.Middleware(tracer, http.DefaultServeMux))
//
// The options allow fine tuning the behavior of the traceMiddleware.
//
// Example:
//
//	  mw := nethttp.Middleware(
//	     tracer,
//	     http.DefaultServeMux,
//	     netottp.OperationNameFunc(func(r *http.Request) string {
//			return "HTTP " + r.Method + ":/api/customers"
//	     }),
//	     nethttp mwSpanObserver(func(sp opentracing.Span, r *http.Request) {
//				sp.SetTag("http.uri", r.URL.EscapedPath())
//			}),
//	  )
func traceMiddleware(tr opentracing.Tracer, h http.Handler, options ...mwOption) http.Handler {
	opts := mwOptions{
		opNameFunc: func(r *http.Request) string {
			return "HTTP " + r.Method
		},
		spanObserver: func(span opentracing.Span, r *http.Request) {},
	}

	for _, opt := range options {
		opt(&opts)
	}

	fn := func(w http.ResponseWriter, r *http.Request) {
		ctx, _ := tr.Extract(opentracing.HTTPHeaders, opentracing.HTTPHeadersCarrier(r.Header))
		sp := tr.StartSpan(opts.opNameFunc(r), ext.RPCServerOption(ctx))
		ext.HTTPMethod.Set(sp, r.Method)
		ext.HTTPUrl.Set(sp, r.URL.String())
		opts.spanObserver(sp, r)

		// set component name, use "net/http" if caller does not specify
		componentName := opts.componentName
		ext.Component.Set(sp, componentName)

		r = r.WithContext(opentracing.ContextWithSpan(r.Context(), sp))

		h.ServeHTTP(w, r)

		if sw, ok := w.(statusWriter); ok {
			ext.HTTPStatusCode.Set(sp, uint16(sw.Status()))
		}
		sp.Finish()
	}
	return http.HandlerFunc(fn)
}

type statusWriter interface {
	Status() int
}
