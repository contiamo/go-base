package queue

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSwitchMetricsServiceName(t *testing.T) {
	t.Run("does not panic when try to register something that matches default", func(t *testing.T) {
		require.NotPanics(t, func() {
			SwitchMetricsServiceName(constLabels[serviceKey])
		})
	})

	t.Run("does not panic when try to register something else", func(t *testing.T) {
		require.NotPanics(t, func() {
			SwitchMetricsServiceName("test")
		})
	})

	t.Run("does not panic when try to register something twice", func(t *testing.T) {
		require.NotPanics(t, func() {
			SwitchMetricsServiceName("test")
			SwitchMetricsServiceName("test")
		})
	})
}
