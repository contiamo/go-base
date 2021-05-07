package parameters

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi"
	"github.com/stretchr/testify/require"
)

func Test_Resolve(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cases := []struct {
		name,
		route,
		url string
		expectedValues map[string]string
	}{
		{
			"Parameter are resolved from the path and query",
			"/{some}/other/{another}",
			"/path1/other/path2?q1=query1&q2=query2",
			map[string]string{
				"some":    "path1",
				"another": "path2",
				"q1":      "query1",
				"q2":      "query2",
			},
		},
		{
			"Path parameter has higher priority",
			"/{some}/other",
			"/path1/other?some=query1",
			map[string]string{
				"some": "path1",
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := chi.NewRouter()
			resolver := NewChiResolver()
			triggered := false
			r.Get(tc.route, func(w http.ResponseWriter, r *http.Request) {
				for key, value := range tc.expectedValues {
					resolved := resolver.Resolve(r, key)
					require.Equal(t, value, resolved)
				}
				triggered = true
			})
			server := httptest.NewServer(r)

			req, err := http.NewRequest(http.MethodGet, server.URL+tc.url, nil)
			require.NoError(t, err)
			req = req.WithContext(ctx)
			resp, err := http.DefaultClient.Do(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			require.True(t, triggered)
		})
	}
}
