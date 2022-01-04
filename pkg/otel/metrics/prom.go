package metrics

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/contiamo/go-base/v4/pkg/otel/component"
	prom "github.com/prometheus/client_golang/prometheus"

	"go.opentelemetry.io/contrib/instrumentation/runtime"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/metric/global"
	"go.opentelemetry.io/otel/sdk/export/metric/aggregation"
	controller "go.opentelemetry.io/otel/sdk/metric/controller/basic"
	processor "go.opentelemetry.io/otel/sdk/metric/processor/basic"
	selector "go.opentelemetry.io/otel/sdk/metric/selector/simple"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
)

// Setup configures the OpenTelemetry Resource and Prometheus metrics exporter.
// Additionally, it configures and sets the component_info metric that allows
// you to monitor the version of the application.
//
// All metrics are registered with the default Prometheus global registry.
func Setup(component component.Info) (*prometheus.Exporter, error) {
	baseResource, err := resource.New(
		context.Background(),
		resource.WithHost(),
		resource.WithAttributes(
			semconv.ServiceVersionKey.String(component.Version),
			semconv.ServiceNameKey.String(component.Name),
		),
		resource.WithFromEnv(),
	)
	if err != nil {
		return nil, err
	}

	config := prometheus.Config{
		Registerer: prom.DefaultRegisterer,
		Gatherer:   prom.DefaultGatherer,
	}
	c := controller.New(
		processor.NewFactory(
			selector.NewWithHistogramDistribution(),
			aggregation.CumulativeTemporalitySelector(),
			processor.WithMemory(true),
		),
		controller.WithResource(baseResource),
	)
	exporter, err := prometheus.New(config, c)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize prometheus exporter: %w", err)
	}
	global.SetMeterProvider(exporter.MeterProvider())

	err = runtime.Start(
		runtime.WithMinimumReadMemStatsInterval(time.Second),
	)
	if err != nil {
		log.Fatalln("failed to start runtime instrumentation:", err)
	}

	meter := global.Meter(component.Name)
	componentInfo, err := meter.NewInt64UpDownCounter(
		"component_info",
		metric.WithDescription("version information of the component"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create component_info metric: %w", err)
	}

	componentInfo.Add(
		context.Background(),
		1,
		attribute.Key("version").String(component.Version),
		attribute.Key("commit").String(component.Commit),
		attribute.Key("name").String(component.Name),
	)

	return exporter, nil
}
