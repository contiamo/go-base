package handlers

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/pkg/errors"

	cerrors "github.com/contiamo/go-base/v3/pkg/errors"
	"github.com/stretchr/testify/require"
)

func TestParse(t *testing.T) {
	t.Run("Returns error when syntax is wrong", func(t *testing.T) {
		h := NewBaseHandler("test", Megabyte, false)
		req := httptest.NewRequest(
			http.MethodGet,
			"/",
			bytes.NewBuffer([]byte("illegal")),
		)
		var s struct{}
		err := h.Parse(req, &s)
		require.Error(t, err)
		require.Equal(t, "Failed to read JSON from the request body: invalid character 'i' looking for beginning of value", err.Error())
	})

	t.Run("Returns error when the payload is too big", func(t *testing.T) {
		h := NewBaseHandler("test", 1, false)
		req := httptest.NewRequest(
			http.MethodGet,
			"/",
			bytes.NewBuffer([]byte(`{"name": "gopher"}`)),
		)
		var s struct {
			Name string `json:"name"`
		}
		err := h.Parse(req, &s)
		require.Error(t, err)
		require.Equal(t, "Failed to read JSON from the request body: unexpected EOF", err.Error())
	})

	t.Run("Returns no error and unmarshalls correctly when syntax is right", func(t *testing.T) {
		h := NewBaseHandler("test", Megabyte, false)
		req := httptest.NewRequest(
			http.MethodGet,
			"/",
			bytes.NewBuffer([]byte(`{"name": "gopher"}`)),
		)
		var s struct {
			Name string `json:"name"`
		}
		err := h.Parse(req, &s)
		require.NoError(t, err)
		require.Equal(t, "gopher", s.Name)
	})
}

func TestWrite(t *testing.T) {
	t.Run("Write the response with a given status code for a structure", func(t *testing.T) {
		h := NewBaseHandler("test", Megabyte, false)
		resp := httptest.NewRecorder()
		h.Write(context.Background(), resp, http.StatusCreated, struct{ Name string }{"some"})
		require.Equal(t, http.StatusCreated, resp.Code)
		require.Equal(t, "application/json", resp.Header().Get("content-type"))
		require.Equal(t, "{\"Name\":\"some\"}\n", resp.Body.String())
	})
}

func TestError(t *testing.T) {
	cases := []struct {
		name      string
		err       error
		debug     bool
		expStatus int
		expBody   string
	}{
		{
			name:      "Returns 501 when ErrNotImplemented",
			err:       cerrors.ErrNotImplemented,
			expStatus: http.StatusNotImplemented,
			expBody: `{"errors":[{"type":"GeneralError","message":"Method is not implemented"}]}
`,
		},
		{
			name:      "Returns 401 when ErrAuthorization",
			err:       cerrors.ErrAuthorization,
			expStatus: http.StatusUnauthorized,
			expBody: `{"errors":[{"type":"GeneralError","message":"User is unauthorized, make sure you've logged in"}]}
`,
		},
		{
			name:      "Returns 403 when ErrPermission",
			err:       cerrors.ErrPermission,
			expStatus: http.StatusForbidden,
			expBody: `{"errors":[{"type":"GeneralError","message":"You don't have required permission to perform this action"}]}
`,
		},
		{
			name: "Returns 422 when ErrUnmarshalling",
			err: errors.Wrap(
				errors.New("unexpected end of file"),
				cerrors.ErrUnmarshalling.Error(),
			),
			expStatus: http.StatusUnprocessableEntity,
			expBody: `{"errors":[{"type":"GeneralError","message":"Failed to read JSON from the request body: unexpected end of file"}]}
`,
		},
		{
			name:      "Returns 404 when ErrNotFound",
			err:       cerrors.ErrNotFound,
			expStatus: http.StatusNotFound,
			expBody: `{"errors":[{"type":"GeneralError","message":"The requested object was not found"}]}
`,
		},
		{
			name:      "Returns 422 and field errors when validation errors",
			err:       cerrors.ValidationErrors{"field": errors.New("terrible")},
			expStatus: http.StatusUnprocessableEntity,
			expBody: `{"errors":[{"type":"FieldError","message":"terrible","key":"field"}]}
`,
		},
		{
			name:      "Returns 500 when unknown error",
			err:       errors.New("terrible"),
			expStatus: http.StatusInternalServerError,
			expBody: `{"errors":[{"type":"GeneralError","message":"Internal server error, please try again later"}]}
`,
		},
		{
			name:      "Returns 500 and error details when unknown error and debug mode",
			err:       errors.New("terrible"),
			debug:     true,
			expStatus: http.StatusInternalServerError,
			expBody: `{"errors":[{"type":"GeneralError","message":"terrible"}]}
`,
		},
		{
			name:      "Returns 200 and empty body when error is nil",
			debug:     true,
			expStatus: http.StatusOK,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h := NewBaseHandler("test", Megabyte, tc.debug)
			resp := httptest.NewRecorder()
			h.Error(context.Background(), resp, tc.err)
			require.Equal(t, tc.expStatus, resp.Code)
			require.Equal(t, tc.expBody, resp.Body.String())
		})
	}
}
