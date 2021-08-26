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

	"github.com/cenkalti/backoff/v4"
	cerrors "github.com/contiamo/go-base/v4/pkg/errors"
	ctesting "github.com/contiamo/go-base/v4/pkg/testing"
	"github.com/contiamo/go-base/v4/pkg/tokens"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

func TestRetryClient(t *testing.T) {
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

		token    string
		tokenErr error

		expResponse interface{}
		expError    error
		expErrorStr string

		expAttempts int
	}{
		// retryable cases
		{
			name:   "Returns error response from the server on 500",
			method: http.MethodGet,
			path:   "/some/path",
			out:    &out,

			token: "tokenSample",

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
			expAttempts: 2,
		},
		{
			name:   "Returns error response from the server on 400",
			method: http.MethodGet,
			path:   "/some/path",
			out:    &out,

			token: "tokenSample",

			serverStatus:   http.StatusBadRequest,
			serverResponse: []byte("some crazy internal stuff"),

			expError: APIError{
				Status: http.StatusBadRequest,
				Header: http.Header{
					"Content-Length": []string{"25"},
					"Content-Type":   []string{"application/json"},
					"Date":           []string{"fixed value"},
				},
				Response: []byte("some crazy internal stuff"),
			},
			expAttempts: 2,
		},
		// cases without retries
		{
			name:   "Returns error response from the server on 409, no retries attempted",
			method: http.MethodGet,
			path:   "/some/path",
			out:    &out,

			token: "tokenSample",

			serverStatus:   http.StatusConflict,
			serverResponse: []byte("conflict"),

			expError: APIError{
				Status: http.StatusConflict,
				Header: http.Header{
					"Content-Length": []string{"8"},
					"Content-Type":   []string{"application/json"},
					"Date":           []string{"fixed value"},
				},
				Response: []byte("conflict"),
			},
			expAttempts: 1,
		},
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

			token: "tokenSample",

			serverStatus:   http.StatusOK,
			serverResponse: ctesting.ToJSONBytes(t, resp),

			expResponse: &response{
				Answer: resp.Answer,
			},
			expAttempts: 1,
		},
		{
			name:   "Returns ErrAuthorization on 401",
			method: http.MethodGet,
			path:   "/some/path",
			out:    &out,

			token:        "tokenSample",
			serverStatus: http.StatusUnauthorized,

			expError:    cerrors.ErrAuthorization,
			expAttempts: 1,
		},
		{
			name:   "Returns ErrPermission on 403",
			method: http.MethodGet,
			path:   "/some/path",
			out:    &out,

			token:        "tokenSample",
			serverStatus: http.StatusForbidden,

			expError:    cerrors.ErrPermission,
			expAttempts: 1,
		},
		{
			name:   "Returns ErrNotFound on 404",
			method: http.MethodGet,
			path:   "/some/path",
			out:    &out,

			token:        "tokenSample",
			serverStatus: http.StatusNotFound,

			expError:    cerrors.ErrNotFound,
			expAttempts: 1,
		},
		{
			name:   "Returns ErrNotImplemented on 501",
			method: http.MethodGet,
			path:   "/some/path",
			out:    &out,

			token:        "tokenSample",
			serverStatus: http.StatusNotImplemented,

			expError:    cerrors.ErrNotImplemented,
			expAttempts: 1,
		},
		{
			name:   "Gets invalid JSON and propagates the error",
			method: http.MethodPost,
			path:   "/some/path",
			out:    &out,

			token:          "tokenSample",
			serverStatus:   http.StatusOK,
			serverResponse: []byte("invalid"),

			expErrorStr: "failed to decode JSON response: invalid character 'i' looking for beginning of value",
			expAttempts: 1,
		},
		{
			name:    "Posts payload with invalid JSON and propagates the error",
			method:  http.MethodPost,
			path:    "/some/path",
			payload: invalidPayload{},
			out:     &out,

			token:          "tokenSample",
			serverStatus:   http.StatusOK,
			serverResponse: ctesting.ToJSONBytes(t, resp),

			expErrorStr: "json: error calling MarshalJSON for type clients.invalidPayload: invalid payload",
			expAttempts: 1,
		},
		// cases with no attempts
		{
			name:   "Propagates the token creator error",
			method: http.MethodGet,
			path:   "/some/path",
			out:    &out,

			serverStatus: http.StatusOK,
			token:        "tokenSample",
			tokenErr:     tokenError,

			expError: tokenError,
		},
	}

	basePath := "/retry-base"

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			out = response{} // reset the value, so make sure the new one is received by the current test case

			var totalAttempts int
			s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				totalAttempts++

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

			tm := &tokens.CreatorMock{
				Err:   tc.tokenErr,
				Token: tc.token,
			}
			tp := TokenProviderFromCreator(tm, "test", tokens.Options{})

			c := NewBaseAPIClient(s.URL+basePath, "X-Request-Token", tp, http.DefaultClient, true)
			c = WithRetry(c, 2, backoff.NewConstantBackOff(500*time.Millisecond))

			require.Equal(t, s.URL+basePath, c.GetBaseURL())

			err := c.DoRequest(ctx, tc.method, tc.path, tc.query, tc.payload, tc.out)
			require.Equal(t, tc.expAttempts, totalAttempts)

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
