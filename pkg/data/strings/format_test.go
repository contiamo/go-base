package strings

import (
	"math"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestHumanReadableByteCount(t *testing.T) {
	cases := []struct {
		name     string
		bytes    int64
		expected string
	}{
		{
			name:     "Converts bytes",
			bytes:    999,
			expected: "999 B",
		},
		{
			name:     "Converts and rounds kilobytes",
			bytes:    1000,
			expected: "1.0 kB",
		},
		{
			name:     "Converts and rounds kilobytes — 2",
			bytes:    1023,
			expected: "1.0 kB",
		},
		{
			name:     "Converts and rounds kilobytes — 3",
			bytes:    1024,
			expected: "1.0 kB",
		},
		{
			name:     "Converts megabytes",
			bytes:    987654321,
			expected: "987.7 MB",
		},
		{
			name:     "Converts very large values",
			bytes:    math.MaxInt64,
			expected: "9.2 EB",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.expected, HumanReadableByteCount(tc.bytes))
		})
	}

}
