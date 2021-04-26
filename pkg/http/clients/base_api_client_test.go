package clients

import (
	"bytes"
	"context"
	"encoding/json"

	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"testing"
	"time"

	cerrors "github.com/contiamo/go-base/v3/pkg/errors"
	ctesting "github.com/contiamo/go-base/v3/pkg/testing"
	"github.com/contiamo/go-base/v3/pkg/tokens"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

type invalidPayload struct {
}

func (p invalidPayload) MarshalJSON() ([]byte, error) {
	return nil, errors.New("invalid payload")
}

func TestBaseAPIClientDoRequest(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	logrus.SetOutput(ioutil.Discard)
	defer logrus.SetOutput(os.Stdout)

	type payload struct {
		String  string `json:"string"`
		Integer int    `json:"integer"`
	}

	type response struct {
		Answer string `json:"answer"`
	}

	var (
		out  response
		resp = response{
			Answer: "everything",
		}
		tokenError = errors.New("token error")
	)

	cases := []struct {
		name string

		method  string
		path    string
		query   url.Values
		payload interface{}
		out     interface{}

		serverStatus   int
		serverResponse []byte

		tokenErr error

		expResponse interface{}
		expError    error
		expErrorStr string
	}{
		{
			name:   "Posts payload, gets response back with 200",
			method: http.MethodPost,
			path:   "/some/path",
			query: url.Values{
				"q1": []string{"v1"},
				"q2": []string{"v2", "v3"},
			},
			payload: payload{
				String:  "some test",
				Integer: 42,
			},
			out: &out,

			serverStatus:   http.StatusOK,
			serverResponse: ctesting.ToJSONBytes(t, resp),

			expResponse: &response{
				Answer: resp.Answer,
			},
		},
		{
			name:   "Posts nothing, gets response back with 200",
			method: http.MethodPost,
			path:   "/some/path",
			out:    &out,

			serverStatus:   http.StatusOK,
			serverResponse: ctesting.ToJSONBytes(t, resp),

			expResponse: &response{
				Answer: resp.Answer,
			},
		},
		{
			name:   "Posts nothing, gets nothing with 204",
			method: http.MethodPost,
			path:   "/some/path",
			out:    &out,

			serverStatus: http.StatusNoContent,

			expResponse: &response{},
		},
		{
			name:   "Gets response with 200",
			method: http.MethodGet,
			path:   "/some/path",
			out:    &out,

			serverStatus:   http.StatusOK,
			serverResponse: ctesting.ToJSONBytes(t, resp),

			expResponse: &response{
				Answer: resp.Answer,
			},
		},
		{
			name:   "Returns ErrAuthorization on 401",
			method: http.MethodGet,
			path:   "/some/path",
			out:    &out,

			serverStatus: http.StatusUnauthorized,

			expError: cerrors.ErrAuthorization,
		},
		{
			name:   "Returns ErrPermission on 403",
			method: http.MethodGet,
			path:   "/some/path",
			out:    &out,

			serverStatus: http.StatusForbidden,

			expError: cerrors.ErrPermission,
		},
		{
			name:   "Returns ErrNotFound on 404",
			method: http.MethodGet,
			path:   "/some/path",
			out:    &out,

			serverStatus: http.StatusNotFound,

			expError: cerrors.ErrNotFound,
		},
		{
			name:   "Returns ErrNotImplemented on 501",
			method: http.MethodGet,
			path:   "/some/path",
			out:    &out,

			serverStatus: http.StatusNotImplemented,

			expError: cerrors.ErrNotImplemented,
		},
		{
			name:   "Returns error response from the server on 500",
			method: http.MethodGet,
			path:   "/some/path",
			out:    &out,

			serverStatus:   http.StatusInternalServerError,
			serverResponse: []byte("some crazy internal stuff"),

			expError: APIError{
				Status: http.StatusInternalServerError,
				Header: http.Header{
					"Content-Length": []string{"25"},
					"Content-Type":   []string{"application/json"},
					"Date":           []string{"fixed value"},
				},
				Response: []byte("some crazy internal stuff"),
			},
		},
		{
			name:   "Propogates the token creator error",
			method: http.MethodGet,
			path:   "/some/path",
			out:    &out,

			serverStatus: http.StatusOK,
			tokenErr:     tokenError,

			expError: tokenError,
		},
		{
			name:    "Posts payload with invalid JSON and propagates the error",
			method:  http.MethodPost,
			path:    "/some/path",
			payload: invalidPayload{},
			out:     &out,

			serverStatus:   http.StatusOK,
			serverResponse: ctesting.ToJSONBytes(t, resp),

			expErrorStr: "json: error calling MarshalJSON for type clients.invalidPayload: invalid payload",
		},
		{
			name:   "Gets invalid JSON and propagates the error",
			method: http.MethodPost,
			path:   "/some/path",
			out:    &out,

			serverStatus:   http.StatusOK,
			serverResponse: []byte("invalid"),

			expErrorStr: "failed to decode JSON response: invalid character 'i' looking for beginning of value",
		},
	}

	token := "tokenSample"
	basePath := "/base"

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			out = response{} // reset the value, so make sure the new one is received by the current test case

			s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				require.Equal(t, basePath+tc.path, r.URL.Path)
				require.Equal(t, tc.query.Encode(), r.URL.Query().Encode())
				require.Equal(t, token, r.Header.Get("X-Request-Token"))
				require.Equal(t, "application/json", r.Header.Get("Content-Type"))

				// ignore all the payload errors, we just need to compare bytes
				payload, _ := ioutil.ReadAll(r.Body)
				payload = bytes.TrimSpace(payload)
				if tc.payload != nil {
					// ignore the serialization error, we compare bytes anyway
					expBytes, _ := json.Marshal(tc.payload)
					require.Equal(t, string(expBytes), string(payload))
				} else {
					require.Empty(t, payload)
				}

				if len(tc.serverResponse) > 0 {
					w.Header().Add("content-type", "application/json")
				}
				w.Header().Add("date", "fixed value")
				w.WriteHeader(tc.serverStatus)
				_, _ = w.Write(tc.serverResponse)
			}))
			defer s.Close()

			cm := &tokens.CreatorMock{
				Err:   tc.tokenErr,
				Token: token,
			}
			c := NewBaseAPIClient(s.URL+basePath, "X-Request-Token", cm, http.DefaultClient, true)
			err := c.DoRequest(ctx, tc.method, tc.path, tc.query, tc.payload, tc.out)
			if tc.expError != nil {
				require.Error(t, err)
				require.Equal(t, tc.expError, errors.Cause(err))
				return
			}
			if tc.expErrorStr != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.expErrorStr)
				return
			}

			require.NoError(t, err)
			require.EqualValues(t, tc.expResponse, tc.out)
		})
	}

}
