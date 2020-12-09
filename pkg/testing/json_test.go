package testing

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestJSONEq(t *testing.T) {

	validCases := []struct {
		name     string
		expected string
		got      string
	}{
		{"valid on empty strings", "", ""},
		{"handles simple reordering of keys", `{"hello": "world", "foo": "bar"}`, `{"foo": "bar", "hello": "world"}`},
	}

	for _, tc := range validCases {
		t.Run(tc.name, func(t *testing.T) {
			JSONEq(t, tc.expected, tc.got)
		})
	}

	failureCases := []struct {
		name     string
		expected string
		got      string
	}{
		{"failure on empty string and non-empty string", "", `{"hello": "world", "foo": "bar"}`},
		{"handles json mismatch", `{"hello": "world"}`, `{"foo": "bar", "hello": "world"}`},
	}

	for _, tc := range failureCases {
		t.Run(tc.name, func(t *testing.T) {
			mt := new(MockT)
			JSONEq(mt, tc.expected, tc.got)

			require.True(t, mt.Failed, "should fail")
		})
	}
}

type MockT struct {
	Failed bool
}

func (t *MockT) FailNow() {
	t.Failed = true
}

func (t *MockT) Errorf(format string, args ...interface{}) {
	_, _ = format, args
}
