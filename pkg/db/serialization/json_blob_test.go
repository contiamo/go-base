package serialization

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestJSONBlobScan(t *testing.T) {
	cases := []struct {
		name     string
		data     interface{}
		src      interface{}
		expected interface{}
		err      string
	}{
		{
			name: "Populates the underlying struct with values when source is byte slice",
			data: &struct{ Name string }{},
			src:  []byte(`{"Name":"value"}`),
			expected: &struct {
				Name string
			}{Name: "value"},
		},
		{
			name: "Populates the underlying struct with values when source is string",
			data: &struct{ Name string }{},
			src:  `{"Name":"value"}`,
			expected: &struct {
				Name string
			}{Name: "value"},
		},
		{
			name:     "Support scanning nil",
			data:     &struct{ Name string }{},
			src:      nil,
			expected: &struct{ Name string }{},
		},
		{
			name: "Returns error when the source is not a valid JSON",
			data: &struct{ Name string }{},
			src:  []byte(`{"Name":invalid`),
			expected: &struct {
				Name string
			}{Name: "value"},
			err: "invalid character 'i' looking for beginning of value",
		},
		{
			name: "Returns error when the source is not a byte slice",
			data: &struct{ Name string }{},
			src:  time.Now(),
			expected: &struct {
				Name string
			}{Name: "value"},
			err: "unknown json object type time.Time",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			b := JSONBlob(tc.data)
			err := b.Scan(tc.src)

			if tc.err != "" {
				require.Error(t, err)
				require.Equal(t, tc.err, err.Error())
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.expected, b.GetData())
		})
	}
}

func TestJSONBlobValue(t *testing.T) {
	t.Run("Returns JSON when the underlying data is valid", func(t *testing.T) {
		b := JSONBlob(struct{ Name string }{Name: "value"})
		bytes, err := b.Value()
		require.NoError(t, err)
		require.Equal(t, []byte(`{"Name":"value"}`), bytes)
	})
}
