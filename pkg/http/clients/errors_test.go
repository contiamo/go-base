package clients

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/contiamo/go-base/v4/pkg/errors"
	ctesting "github.com/contiamo/go-base/v4/pkg/testing"
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

			t.Run("test GetStatusCode returns correct status code", func(t *testing.T) {
				require.Equal(t, tc.e.Status, GetStatusCode(tc.e))
			})
		})
	}
}

func TestGetStatusCode(t *testing.T) {
	cases := []struct {
		name     string
		err      error
		expected int
	}{
		{
			name:     "empty error returns 200",
			expected: 200,
		},
		{
			name:     "base error not found returns 404",
			err:      errors.ErrNotFound,
			expected: http.StatusNotFound,
		},
		{
			name:     "base ErrAuthorization returns 401",
			err:      errors.ErrAuthorization,
			expected: http.StatusUnauthorized,
		},
		{
			name:     "base ErrPermission returns 403",
			err:      errors.ErrPermission,
			expected: http.StatusForbidden,
		},
		{
			name:     "base ErrNotImplemented returns 501",
			err:      errors.ErrNotImplemented,
			expected: http.StatusNotImplemented,
		},
		{
			name:     "unknown error returns 0",
			err:      fmt.Errorf("foo"),
			expected: 0,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.expected, GetStatusCode(tc.err))
		})
	}
}
