package tracing

import (
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	jaeger "github.com/uber/jaeger-client-go"
)

var testJaegerServer = fmt.Sprintf("%s:%d", jaeger.DefaultUDPSpanServerHost, jaeger.DefaultUDPSpanServerPort)

func Test_getConfig(t *testing.T) {
	cases := []struct {
		name           string
		server         string
		varname        string
		expectedServer string
	}{
		{"set local agent server from ENV", "example.com", "JAEGER_AGENT_HOST", "example.com:6831"},
		{"set remote agent server from ENV", "http://example.com", "JAEGER_ENDPOINT", "http://example.com"},
		{"use default local agent from env", "", "", testJaegerServer},
	}

	currentEnvEndpoint := os.Getenv("JAEGER_ENDPOINT")
	currentEnvHost := os.Getenv("JAEGER_AGENT_HOST")
	defer os.Setenv("JAEGER_ENDPOINT", currentEnvEndpoint)
	defer os.Setenv("JAEGER_AGENT_HOST", currentEnvHost)

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.varname != "" {
				os.Setenv(tc.varname, tc.server)
				defer os.Unsetenv(tc.varname)
			}

			cfg, err := getConfig("test-app")
			require.NoError(t, err)

			require.Equal(t, cfg.ServiceName, "test-app")
			if tc.varname == "JAEGER_ENDPOINT" {
				require.Equal(t, tc.expectedServer, cfg.Reporter.CollectorEndpoint)
			} else {
				require.Equal(t, tc.expectedServer, cfg.Reporter.LocalAgentHostPort)
			}
		})
	}
}
