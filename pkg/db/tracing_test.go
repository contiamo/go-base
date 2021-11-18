package db

import (
	"testing"

	"github.com/opentracing/opentracing-go"
	"github.com/stretchr/testify/require"
)

type spanMock struct {
	opentracing.Span
	key   string
	value string
}

//nolint:errcheck,forcetypeassert // it should panic if not a string
func (s *spanMock) LogKV(alternatingKeyValues ...interface{}) {
	s.key = alternatingKeyValues[0].(string)
	s.value = alternatingKeyValues[1].(string)
}

func TestWithTrimmedQuery(t *testing.T) {
	tdb, ok := WrapWithTracing(nil).(*traceableDB)
	require.True(t, ok, "wrong type")

	t.Run("no trimming set", func(t *testing.T) {
		s := &spanMock{}
		q := "SELECT very long long table"
		tdb.logQuery(s, q)
		require.Equal(t, "sql", s.key)
		require.Equal(t, q, s.value)
	})

	t.Run("trimming set", func(t *testing.T) {
		s := &spanMock{}

		cases := []struct {
			name     string
			query    string
			limit    uint
			expKey   string
			expValue string
		}{
			{
				name:     "does not log when set to 0",
				query:    "some query",
				limit:    0,
				expKey:   "",
				expValue: "",
			},
			{
				name:     "trims when set to a positive number and query is long",
				query:    "some query",
				limit:    3,
				expKey:   "sql",
				expValue: "som...",
			},
			{
				name:     "does not trim when set to a positive number and query is short",
				query:    "some",
				limit:    128,
				expKey:   "sql",
				expValue: "some",
			},
			{
				name:     "does not trim when set to length of query",
				query:    "some",
				limit:    4,
				expKey:   "sql",
				expValue: "some",
			},
		}

		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				tdb, ok := tdb.WithTrimmedQuery(tc.limit).(*traceableDB)
				require.True(t, ok, "wrong type")
				tdb.logQuery(s, tc.query)
				require.Equal(t, tc.expKey, s.key)
				require.Equal(t, tc.expValue, s.value)
			})
		}
	})
}
