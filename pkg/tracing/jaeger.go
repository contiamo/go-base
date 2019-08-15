package tracing

import (
	"fmt"
	"time"

	opentracing "github.com/opentracing/opentracing-go"
	jaeger "github.com/uber/jaeger-client-go"
	"github.com/uber/jaeger-client-go/config"
)

var defaultJaegerServer = fmt.Sprintf("%s:%d", jaeger.DefaultUDPSpanServerHost, jaeger.DefaultUDPSpanServerPort)

// InitJaeger asserts that the global tracer is initialized.
//
// This will read the configuration from the "JAEGER_*"" environment variables.
// Overriding the empty values with the supplied app value.  If a
// sampler type is not configured via the environment variables, then InitJaeger
// will be configured with the constant sampler.
func InitJaeger(app string) error {
	global := opentracing.GlobalTracer()
	if _, ok := global.(opentracing.NoopTracer); ok {

		cfg, err := getConfig(app)
		if err != nil {
			return err
		}

		_, err = cfg.InitGlobalTracer(app, config.Logger(jaeger.StdLogger))
		if err != nil {
			return err
		}
	}
	return nil
}

func getConfig(app string) (*config.Configuration, error) {
	cfg, err := config.FromEnv()
	if err != nil {
		return nil, err
	}
	if cfg.ServiceName == "" {
		cfg.ServiceName = app
	}

	if cfg.Sampler.Type == "" {
		cfg.Sampler.Type = "const"
		cfg.Sampler.Param = 1
	}

	if cfg.Reporter.BufferFlushInterval == 0 {
		cfg.Reporter.BufferFlushInterval = 1 * time.Second
	}

	return cfg, nil
}
