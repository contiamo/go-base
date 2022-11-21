package testing

import (
	"github.com/stretchr/testify/require"
)

type tHelper interface {
	Helper()
}

// JSONEq asserts that two JSON strings are equivalent or both empty.
//
//	require.JSONEq(t, "", "")
//	or
//	require.JSONEq(t, `{"hello": "world", "foo": "bar"}`, `{"foo": "bar", "hello": "world"}`)
func JSONEq(t require.TestingT, expected string, actual string, msgAndArgs ...interface{}) {
	if h, ok := t.(tHelper); ok {
		h.Helper()
	}
	// handle empty json
	if expected == actual && expected == "" {
		return
	}

	require.JSONEq(t, expected, actual, msgAndArgs...)
}
