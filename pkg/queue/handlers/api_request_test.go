package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"sort"
	"testing"
	"time"

	"github.com/contiamo/go-base/v4/pkg/queue"
	test "github.com/contiamo/go-base/v4/pkg/testing"
	"github.com/contiamo/go-base/v4/pkg/tokens"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"
)

// we can't compare `Duration`
type httpRequestProgressToCompare struct {
	Stage          APIRequestStage `json:"stage,omitempty"`
	ReturnedStatus *int            `json:"returnedStatus,omitempty"`
	ReturnedBody   *string         `json:"returnedBody,omitempty"`
	ErrorMessage   *string         `json:"errorMessage,omitempty"`
}

func toComparableAPIRequestProgress(val APIRequestProgress) httpRequestProgressToCompare {
	return httpRequestProgressToCompare{
		Stage:          val.Stage,
		ReturnedStatus: val.ReturnedStatus,
		ReturnedBody:   val.ReturnedBody,
		ErrorMessage:   val.ErrorMessage,
	}
}

func TestAPIRequestHandlerProcess(t *testing.T) {
	defer verifyLeak(t)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	logrus.SetOutput(ioutil.Discard)
	defer logrus.SetOutput(os.Stdout)

	var headers []string

	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-time.After(100 * time.Millisecond) // imitate the network lag

		buf, err := ioutil.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.Header().Set("content-type", "application/json")
		w.WriteHeader(http.StatusCreated)

		for name, values := range r.Header {
			headers = append(headers, fmt.Sprintf("%s: %v", name, values))
		}
		sort.Strings(headers)

		_, _ = w.Write([]byte{'{'})
		_, _ = w.Write([]byte(`"incoming":"` + string(buf) + `"`))
		_, _ = w.Write([]byte(`,"outcoming":"message"`))
		_, _ = w.Write([]byte{'}'})
	}))
	defer s.Close()

	cases := []struct {
		name        string
		spec        APIRequestTaskSpec
		token       string
		tokenError  error
		expError    string
		expHeaders  []string
		expProgress APIRequestProgress
	}{
		{
			name: "does the authorized request to a valid endpoint with valid parameters",
			spec: APIRequestTaskSpec{
				Method:      http.MethodPost,
				URL:         s.URL,
				RequestBody: "task request",
				RequestHeaders: map[string]string{
					"X-Some":    "X-Value",
					"X-Another": "X-Some",
				},
				Authorized:     true,
				ExpectedStatus: http.StatusCreated,
			},
			token: "token value",
			expHeaders: []string{
				"Accept-Encoding: [gzip]",
				"Auth: [token value]",
				"Content-Length: [12]",
				"User-Agent: [Contiamo API Request Task]",
				"X-Another: [X-Some]",
				"X-Some: [X-Value]",
			},
			expProgress: APIRequestProgress{
				Stage:          RequestResponse,
				ReturnedStatus: intP(http.StatusCreated),
				ReturnedBody:   strP(`{"incoming":"task request","outcoming":"message"}`),
			},
		},
		{
			name: "does the unauthorized request to a valid endpoint with valid parameters",
			spec: APIRequestTaskSpec{
				Method:      http.MethodPost,
				URL:         s.URL,
				RequestBody: "task request",
				RequestHeaders: map[string]string{
					"X-Some":    "X-Value",
					"X-Another": "X-Some",
				},
				Authorized:     false,
				ExpectedStatus: http.StatusCreated,
			},
			token: "token value",
			expHeaders: []string{
				"Accept-Encoding: [gzip]",
				"Content-Length: [12]",
				"User-Agent: [Contiamo API Request Task]",
				"X-Another: [X-Some]",
				"X-Some: [X-Value]",
			},
			expProgress: APIRequestProgress{
				Stage:          RequestResponse,
				ReturnedStatus: intP(http.StatusCreated),
				ReturnedBody:   strP(`{"incoming":"task request","outcoming":"message"}`),
			},
		},
		{
			name: "does the unauthorized request without headers to a valid endpoint with valid parameters",
			spec: APIRequestTaskSpec{
				Method:         http.MethodPost,
				URL:            s.URL,
				RequestBody:    "task request",
				ExpectedStatus: http.StatusCreated,
			},
			expHeaders: []string{
				"Accept-Encoding: [gzip]",
				"Content-Length: [12]",
				"User-Agent: [Contiamo API Request Task]",
			},
			expProgress: APIRequestProgress{
				Stage:          RequestResponse,
				ReturnedStatus: intP(http.StatusCreated),
				ReturnedBody:   strP(`{"incoming":"task request","outcoming":"message"}`),
			},
		},
		{
			name: "does the unauthorized request without headers or body to a valid endpoint with valid parameters",
			spec: APIRequestTaskSpec{
				Method:         http.MethodPost,
				URL:            s.URL,
				ExpectedStatus: http.StatusCreated,
			},
			expHeaders: []string{
				"Accept-Encoding: [gzip]",
				"Content-Length: [0]",
				"User-Agent: [Contiamo API Request Task]",
			},
			expProgress: APIRequestProgress{
				Stage:          RequestResponse,
				ReturnedStatus: intP(http.StatusCreated),
				ReturnedBody:   strP(`{"incoming":"","outcoming":"message"}`),
			},
		},
		{
			name: "fails when the response status does not match",
			spec: APIRequestTaskSpec{
				Method:         http.MethodGet,
				URL:            s.URL,
				ExpectedStatus: http.StatusOK,
			},
			expHeaders: []string{
				"Accept-Encoding: [gzip]",
				"User-Agent: [Contiamo API Request Task]",
			},
			expProgress: APIRequestProgress{
				Stage:          RequestResponse,
				ReturnedStatus: intP(http.StatusCreated),
				ReturnedBody:   strP(`{"incoming":"","outcoming":"message"}`),
				ErrorMessage:   strP("expected status 200 but got 201"),
			},
			expError: "expected status 200 but got 201",
		},
		{
			name: "fails when the target URL is invalid",
			spec: APIRequestTaskSpec{
				Method:         http.MethodPost,
				URL:            string([]byte{0x7f}),
				ExpectedStatus: http.StatusCreated,
			},
			expProgress: APIRequestProgress{
				Stage:        RequestPreparing,
				ErrorMessage: strP("parse \"\\u007f\": net/url: invalid control character in URL"),
			},
			expError: "parse \"\\u007f\": net/url: invalid control character in URL",
		},
		{
			name: "fails when the target URL points nowhere",
			spec: APIRequestTaskSpec{
				Method:         http.MethodPost,
				URL:            "javascript://wrong",
				ExpectedStatus: http.StatusCreated,
			},
			expProgress: APIRequestProgress{
				Stage:        RequestPending,
				ErrorMessage: strP("Post \"javascript://wrong\": unsupported protocol scheme \"javascript\""),
			},
			expError: "Post \"javascript://wrong\": unsupported protocol scheme \"javascript\"",
		},
		{
			name: "fails when the method is invalid",
			spec: APIRequestTaskSpec{
				Method:         "WR ONG",
				URL:            s.URL,
				ExpectedStatus: http.StatusCreated,
			},
			expProgress: APIRequestProgress{
				Stage:        RequestPreparing,
				ErrorMessage: strP("net/http: invalid method \"WR ONG\""),
			},
			expError: "net/http: invalid method \"WR ONG\"",
		},
		{
			name: "fails when the token creator fails",
			spec: APIRequestTaskSpec{
				Method:         http.MethodGet,
				URL:            s.URL,
				ExpectedStatus: http.StatusCreated,
				Authorized:     true,
			},
			tokenError: errors.New("oops"),
			expProgress: APIRequestProgress{
				Stage:        RequestPreparing,
				ErrorMessage: strP("oops"),
			},
			expError: "oops",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			hrh := NewAPIRequestHandler(
				"auth",
				&tokens.CreatorMock{
					Err:   tc.tokenError,
					Token: tc.token,
				},
				http.DefaultClient,
			)

			// collect heartbeats with the status
			var progress queue.Progress
			beats := make(chan queue.Progress)
			ready := make(chan bool)
			go func() {
				for b := range beats {
					progress = b
				}
				ready <- true
			}()

			task := queue.Task{
				TaskBase: queue.TaskBase{
					Spec: test.ToJSONBytes(t, tc.spec),
				},
			}

			headers = nil
			err := hrh.Process(ctx, task, beats)

			<-ready

			var progressStruct APIRequestProgress
			progressErr := json.Unmarshal(progress, &progressStruct)
			require.NoError(t, progressErr)

			require.EqualValues(t,
				toComparableAPIRequestProgress(tc.expProgress),
				toComparableAPIRequestProgress(progressStruct),
			)

			if tc.expProgress.Stage != RequestPreparing {
				require.NotNil(t, progressStruct.Duration)
			}

			require.Equal(t, tc.expHeaders, headers)

			if tc.expError != "" {
				require.Error(t, err)
				require.Equal(t, tc.expError, err.Error())
				return
			}
			require.NoError(t, err)
		})
	}

	t.Run("returns error if response content type is not JSON", func(t *testing.T) {
		s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("{}"))
		}))
		defer s.Close()

		hrh := NewAPIRequestHandler(
			"auth",
			&tokens.CreatorMock{},
			http.DefaultClient,
		)

		// collect heartbeats with the status
		var progress []APIRequestProgress
		beats := make(chan queue.Progress)
		ready := make(chan bool)

		go func() {
			for b := range beats {
				var p APIRequestProgress
				err := json.Unmarshal(b, &p)
				require.NoError(t, err)

				// this field is not deterministic, so we remove the value
				p.Duration = nil
				progress = append(progress, p)
			}
			ready <- true
		}()

		spec := APIRequestTaskSpec{
			Method:         http.MethodPost,
			URL:            s.URL,
			ExpectedStatus: http.StatusOK,
		}

		task := queue.Task{
			TaskBase: queue.TaskBase{
				Spec: test.ToJSONBytes(t, spec),
			},
		}

		err := hrh.Process(ctx, task, beats)
		require.Error(t, err)
		require.Equal(t, "unexpected response content type, expected JSON, got `text/plain; charset=utf-8`", err.Error())

		<-ready
	})

	t.Run("returns error if response content type is JSON but response is invalid JSON", func(t *testing.T) {
		s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("content-type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("invalid JSON"))
		}))
		defer s.Close()

		hrh := NewAPIRequestHandler(
			"auth",
			&tokens.CreatorMock{},
			http.DefaultClient,
		)

		// collect heartbeats with the status
		var progress []APIRequestProgress
		beats := make(chan queue.Progress)
		ready := make(chan bool)

		go func() {
			for b := range beats {
				var p APIRequestProgress
				err := json.Unmarshal(b, &p)
				require.NoError(t, err)

				// this field is not deterministic, so we remove the value
				p.Duration = nil
				progress = append(progress, p)
			}
			ready <- true
		}()

		spec := APIRequestTaskSpec{
			Method:         http.MethodPost,
			URL:            s.URL,
			ExpectedStatus: http.StatusOK,
		}

		task := queue.Task{
			TaskBase: queue.TaskBase{
				Spec: test.ToJSONBytes(t, spec),
			},
		}

		err := hrh.Process(ctx, task, beats)
		require.Error(t, err)
		require.Equal(t, "invalid character 'i' looking for beginning of value", err.Error())

		<-ready
	})

	t.Run("returns no error if response content type is JSON but response is empty", func(t *testing.T) {
		s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("content-type", "application/json")
			w.WriteHeader(http.StatusOK)
		}))
		defer s.Close()

		hrh := NewAPIRequestHandler(
			"auth",
			&tokens.CreatorMock{},
			http.DefaultClient,
		)

		// collect heartbeats with the status
		var progress []APIRequestProgress
		beats := make(chan queue.Progress)
		ready := make(chan bool)

		go func() {
			for b := range beats {
				var p APIRequestProgress
				err := json.Unmarshal(b, &p)
				require.NoError(t, err)

				// this field is not deterministic, so we remove the value
				p.Duration = nil
				progress = append(progress, p)
			}
			ready <- true
		}()

		spec := APIRequestTaskSpec{
			Method:         http.MethodPost,
			URL:            s.URL,
			ExpectedStatus: http.StatusOK,
		}

		task := queue.Task{
			TaskBase: queue.TaskBase{
				Spec: test.ToJSONBytes(t, spec),
			},
		}

		err := hrh.Process(ctx, task, beats)
		require.NoError(t, err)

		<-ready
	})

	t.Run("supports partial results as multiple JSON objects and sends progres for them", func(t *testing.T) {
		responses := []string{
			`{"one":"value1"}`,
			`{"two":"value2"}`,
			`{"three":"value3"}`,
		}

		s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			<-time.After(100 * time.Millisecond) // imitate the network lag
			w.Header().Set("content-type", "application/json")
			w.WriteHeader(http.StatusOK)
			for _, resp := range responses {
				_, _ = w.Write([]byte(resp))
			}
		}))
		defer s.Close()

		hrh := NewAPIRequestHandler(
			"auth",
			&tokens.CreatorMock{},
			http.DefaultClient,
		)

		// collect heartbeats with the status
		var progress []APIRequestProgress
		beats := make(chan queue.Progress)
		ready := make(chan bool)

		go func() {
			for b := range beats {
				var p APIRequestProgress
				err := json.Unmarshal(b, &p)
				require.NoError(t, err)

				// this field is not deterministic, so we remove the value
				p.Duration = nil
				progress = append(progress, p)
			}
			ready <- true
		}()

		spec := APIRequestTaskSpec{
			Method:         http.MethodPost,
			URL:            s.URL,
			ExpectedStatus: http.StatusOK,
		}

		task := queue.Task{
			TaskBase: queue.TaskBase{
				Spec: test.ToJSONBytes(t, spec),
			},
		}

		err := hrh.Process(ctx, task, beats)
		require.NoError(t, err)

		<-ready

		require.Equal(
			t,
			[]APIRequestProgress{
				{
					Stage: RequestPreparing,
				},
				{
					Stage: RequestPending,
				},
				{
					Stage:          RequestResponse,
					ReturnedStatus: intP(http.StatusOK),
				},
				{
					Stage:          RequestResponse,
					ReturnedStatus: intP(http.StatusOK),
					ReturnedBody:   strP("{\"one\":\"value1\"}"),
				},
				{
					Stage:          RequestResponse,
					ReturnedStatus: intP(http.StatusOK),
					ReturnedBody:   strP("{\"two\":\"value2\"}"),
				},
				{
					Stage:          RequestResponse,
					ReturnedStatus: intP(http.StatusOK),
					ReturnedBody:   strP("{\"three\":\"value3\"}"),
				},
				// here is the final one that is supposed to contain the duration
				{
					Stage:          RequestResponse,
					ReturnedStatus: intP(http.StatusOK),
					ReturnedBody:   strP("{\"three\":\"value3\"}"),
				},
			},
			progress,
		)
	})
}

func intP(n int) *int {
	return &n
}
func strP(s string) *string {
	return &s
}

func verifyLeak(t *testing.T) {
	_, exists := os.LookupEnv("CI")
	if exists {
		return
	}
	goleak.VerifyNone(t)
}
