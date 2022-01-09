package tracing

import (
	"context"
	"log"
	"os"
	"strings"
	"time"

	"github.com/contiamo/go-base/v4/pkg/otel/component"
	"go.opentelemetry.io/contrib/detectors/aws/eks"
	"go.opentelemetry.io/contrib/propagators/aws/xray"
	jaegerprop "go.opentelemetry.io/contrib/propagators/jaeger"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/jaeger"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
)

// Exporter is the the name of the exporter to use.
type Exporter string

const (
	JaegerExporter   Exporter = "jaeger"
	LogExporter      Exporter = "log"
	OTLPExporter     Exporter = "otlp"
	DisabledExporter Exporter = "disabled"
)

const (
	otelEnvPropagators            = "OTEL_PROPAGATORS"
	otelEnvTraceSExporter         = "OTEL_TRACES_EXPORTER"
	otelEnvExporterLogPrettyPrint = "OTEL_EXPORTER_LOG_PRETTY_PRINT"
	otelEnvExporterLogTimestamps  = "OTEL_EXPORTER_LOG_TIMESTAMPS"
	otelEnvInEKS                  = "OTEL_DETECTOR_EKS"
	otelEnvServiceName            = "OTEL_SERVICE_NAME"
)

type Shutdown func(context.Context)

// Provider returns an OpenTelemetry TracerProvider configured to use
// the Jaeger exporter that will send spans to the provided url. The returned
// TracerProvider will also use a Resource configured with all the information
// about the application.
func Provider(ctx context.Context, component component.Info) (shutdown Shutdown, err error) {
	exporter := Exporter(get(otelEnvTraceSExporter, string(DisabledExporter)))

	var exp trace.TracerProviderOption
	switch exporter {
	case JaegerExporter:
		// configure the collector from the env variables,
		// OTEL_EXPORTER_JAEGER_ENDPOINT/USER/PASSWORD
		// see: https://github.com/open-telemetry/opentelemetry-go/tree/main/exporters/jaeger
		j, e := jaeger.New(jaeger.WithCollectorEndpoint())
		exp, err = trace.WithBatcher(j), e
	case LogExporter:
		w := os.Stdout
		opts := []stdouttrace.Option{stdouttrace.WithWriter(w)}
		if truthyEnv(otelEnvExporterLogPrettyPrint) {
			opts = append(opts, stdouttrace.WithPrettyPrint())
		}
		if !truthyEnv(otelEnvExporterLogTimestamps) {
			opts = append(opts, stdouttrace.WithoutTimestamps())
		}

		s, e := stdouttrace.New(opts...)
		exp, err = trace.WithSyncer(s), e
	case OTLPExporter:
		// find available env variables for configuration
		// see: https://github.com/open-telemetry/opentelemetry-go/tree/main/exporters/otlp/otlptrace#environment-variables
		kind := get("OTEL_EXPORTER_OTLP_PROTOCOL", "grpc")

		var client otlptrace.Client
		switch kind {
		case "grpc":
			client = otlptracegrpc.NewClient()
		case "http":
			client = otlptracehttp.NewClient()
		}
		o, e := otlptrace.New(ctx, client)
		exp, err = trace.WithBatcher(o), e
	default:
		log.Println("tracing disabled")
		// We explicitly DO NOT set the global TracerProvider using otel.SetTracerProvider().
		// The unset TracerProvider returns a "non-recording" span, but still passes through context.
		// return no-op shutdown function
		return func(_ context.Context) {}, nil
	}
	if err != nil {
		return nil, err
	}

	propagators := strings.ToLower(get(otelEnvPropagators, "tracecontext,baggage"))
	otel.SetTextMapPropagator(
		propagation.NewCompositeTextMapPropagator(withPropagators(propagators)...),
	)

	res, err := resource.New(
		context.Background(),
		resource.WithFromEnv(),
		resource.WithHost(),
		resource.WithOS(),
		resource.WithTelemetrySDK(),
		resource.WithAttributes(
			semconv.ServiceVersionKey.String(component.Version),
			attribute.String("service.commit", component.Commit),
			semconv.ServiceNameKey.String(get(otelEnvServiceName, component.Name)),
		),
	)
	if err != nil {
		return nil, err
	}

	if truthyEnv(otelEnvInEKS) {
		awsResourceDetector := eks.NewResourceDetector()
		awsResource, err := awsResourceDetector.Detect(ctx)
		if err != nil {
			return nil, err
		}

		res, err = resource.Merge(res, awsResource)
		if err != nil {
			return nil, err
		}
	}

	provider := trace.NewTracerProvider(
		exp,
		trace.WithResource(res),
		trace.WithSampler(trace.AlwaysSample()),
		trace.WithIDGenerator(idGenerator(propagators)),
	)

	// Register our TracerProvider as the global so any imported
	// instrumentation in the future will default to using it.
	otel.SetTracerProvider(provider)

	shutdown = func(ctx context.Context) {
		// Do not let the application hang forever when it is shutdown.
		ctx, cancel := context.WithTimeout(ctx, time.Second*5)
		defer cancel()

		err := provider.Shutdown(ctx)
		if err != nil {
			log.Printf("failed to shutdown tracing provider: %v", err)
		}
	}

	return shutdown, nil
}

func truthyEnv(name string) bool {
	value, ok := os.LookupEnv(name)
	if !ok {
		return false
	}

	switch value {
	case "true", "1", "yes", "on":
		return true
	default:
		return false
	}
}

func get(name, defaultValue string) string {
	value, ok := os.LookupEnv(name)
	if !ok {
		return defaultValue
	}
	return value
}

func withPropagators(propagators string) []propagation.TextMapPropagator {
	out := []propagation.TextMapPropagator{}

	if strings.Contains(propagators, "tracecontext") {
		out = append(out, propagation.TraceContext{})
	}

	if strings.Contains(propagators, "jaeger") {
		out = append(out, jaegerprop.Jaeger{})
	}

	if strings.Contains(propagators, "xray") {
		out = append(out, xray.Propagator{})
	}

	if strings.Contains(propagators, "baggage") {
		out = append(out, propagation.Baggage{})
	}

	return out
}

// idGenerator determines the trace ID generator based on the propagators list.
// It will be non-nil, if the propagators contains XRay. Otherwise it will be nil
// which results in the default the default random number generator
func idGenerator(propagators string) trace.IDGenerator {
	if strings.Contains(propagators, "xray") {
		return xray.NewIDGenerator()
	}
	return nil
}
