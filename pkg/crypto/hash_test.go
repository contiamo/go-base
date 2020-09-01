package crypto

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestHash(t *testing.T) {
	cases := []struct {
		name           string
		input          []interface{}
		expectedOutput string
		expectedError  error
	}{
		{
			name:           "hash a string",
			input:          []interface{}{"foobar"},
			expectedOutput: "09234807e4af85f17c66b48ee3bca89dffd1f1233659f9f940a2b17b0b8c6bc5",
		},
		{
			name:           "reproducable results",
			input:          []interface{}{"foobar"},
			expectedOutput: "09234807e4af85f17c66b48ee3bca89dffd1f1233659f9f940a2b17b0b8c6bc5",
		},
		{
			name:           "something else",
			input:          []interface{}{"foobarbaz"},
			expectedOutput: "369972cd3fda2b1e239bf114f4c2c65115b05fee2e4e5b2ae19ce7b5d757c572",
		},
		{
			name:           "hash multiple strings",
			input:          []interface{}{"foo", "bar"},
			expectedOutput: "09234807e4af85f17c66b48ee3bca89dffd1f1233659f9f940a2b17b0b8c6bc5",
		},
		{
			name:           "hash reader",
			input:          []interface{}{strings.NewReader("foobar")},
			expectedOutput: "09234807e4af85f17c66b48ee3bca89dffd1f1233659f9f940a2b17b0b8c6bc5",
		},
		{
			name:           "hash bytes",
			input:          []interface{}{[]byte("foobar")},
			expectedOutput: "09234807e4af85f17c66b48ee3bca89dffd1f1233659f9f940a2b17b0b8c6bc5",
		},
		{
			name:           "hash complex object",
			input:          []interface{}{struct{ foo map[string]interface{} }{foo: map[string]interface{}{"a": 123}}},
			expectedOutput: "d0a1b2af1705c1b8495b00145082ef7470384e62ac1c4d9b9cdbbe0476c28f8c",
		},
		{
			name:           "hash multiple different things",
			input:          []interface{}{"f", strings.NewReader("oo"), []byte("bar")},
			expectedOutput: "09234807e4af85f17c66b48ee3bca89dffd1f1233659f9f940a2b17b0b8c6bc5",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			hash, err := HashToString(tc.input...)
			require.Equal(t, tc.expectedOutput, hash)
			require.Equal(t, tc.expectedError, err)
		})
	}
}
