package parameters

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNormalizePagination(t *testing.T) {
	cases := []struct {
		name     string
		number   string
		size     string
		expected Page
	}{
		{
			name:   "Returns default values when parameters are not numbers",
			number: "NANNANNAN",
			size:   "NANNANNAN",
			expected: Page{
				Number: 1,
				Size:   defaultPageSize,
			},
		},
		{
			name: "Returns default values when parameters are not set",
			expected: Page{
				Number: 1,
				Size:   defaultPageSize,
			},
		},
		{
			name:   "Returns 1st page when number is out of range",
			number: "-1",
			size:   "30",
			expected: Page{
				Number: 1,
				Size:   30,
			},
		},
		{
			name:   "Returns minimal page size when size is too small",
			number: "5",
			size:   "1",
			expected: Page{
				Number: 5,
				Size:   minPageSize,
			},
		},
		{
			name:   "Returns maximal page size when size is too big",
			number: "5",
			size:   "999",
			expected: Page{
				Number: 5,
				Size:   maxPageSize,
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.expected, NormalizePagination(tc.number, tc.size))
		})
	}
}
