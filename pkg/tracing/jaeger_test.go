package tracing

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_getConfig(t *testing.T) {

	cases := []struct {
		name           string
		server         string
		serverEnv      string
		expectedServer string
	}{
		{"set server from config", "example.com:6831", "", "example.com:6831"},
		{"set server from ENV", "", "example.com", "example.com:6831"},
	}

	currentEnvHost := os.Getenv("JAEGER_AGENT_HOST")
	currentEnvPort := os.Getenv("JAEGER_AGENT_PORT")
	defer os.Setenv("JAEGER_AGENT_HOST", currentEnvHost)
	defer os.Setenv("JAEGER_AGENT_PORT", currentEnvPort)

	for _, tc := range cases {
		os.Unsetenv("JAEGER_AGENT_HOST")
		os.Unsetenv("JAEGER_AGENT_PORT")

		t.Run(tc.name, func(t *testing.T) {
			os.Setenv("JAEGER_AGENT_HOST", tc.serverEnv)
			cfg, err := getConfig(tc.server, "test-app")
			require.NoError(t, err)

			cfg.Reporter.LocalAgentHostPort = tc.expectedServer
			cfg.ServiceName = "test-app"
		})
	}
}
