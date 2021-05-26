package clients

import (
	"net/http"
	"testing"

	ctesting "github.com/contiamo/go-base/v3/pkg/testing"
	"github.com/stretchr/testify/require"
)

func TestAPIErrorError(t *testing.T) {
	cases := []struct {
		name       string
		e          APIError
		expMessage string
	}{
		{
			name: "Returns status text and no errors",
			e: APIError{
				Status: http.StatusBadRequest,
			},
			expMessage: http.StatusText(http.StatusBadRequest),
		},
		{
			name: "Returns status code if it's not standard",
			e: APIError{
				Status: 522,
			},
			expMessage: "522",
		},
		{
			name: "Returns status text and errors",
			e: APIError{
				Status: http.StatusBadRequest,
				Response: ctesting.ToJSONBytes(t, map[string]interface{}{
					"errors": []map[string]string{
						{
							"message": "general error message",
						},
						{
							"message": "second general error message",
						},
						{
							"key":     "field1",
							"message": "field1 is wrong",
						},
						{
							"key":     "field2",
							"message": "field2 is wrong",
						},
					},
				}),
			},
			expMessage: "Bad Request; general error message; second general error message; field1: field1 is wrong; field2: field2 is wrong.",
		},
		{
			name: "Returns 522 and errors including the special connectivity errors",
			e: APIError{
				Status: 522,
				Response: ctesting.ToJSONBytes(t, map[string]interface{}{
					"errors": []map[string]string{
						{
							"message": "general error message",
						},
						{
							"message": "second general error message",
						},
						{
							"key":     "field1",
							"message": "field1 is wrong",
						},
						{
							"key":     "field2",
							"message": "field2 is wrong",
						},
					},
				}),
			},
			expMessage: "522; connection: general error message. second general error message; field1: field1 is wrong; field2: field2 is wrong.",
		},
		{
			name: "Returns 522 and concatenates connectivity errors",
			e: APIError{
				Status: 522,
				Response: ctesting.ToJSONBytes(t, map[string]interface{}{
					"errors": []map[string]string{
						{
							"message": "first connection issue",
						},
						{
							"message": "second connection issue",
						},
						{
							"key":     "connection",
							"message": "another connection issue",
						},
						{
							"key":     "field1",
							"message": "field1 is wrong",
						},
					},
				}),
			},
			expMessage: "522; connection: first connection issue. second connection issue. another connection issue; field1: field1 is wrong.",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.expMessage, tc.e.Error())
		})
	}
}
