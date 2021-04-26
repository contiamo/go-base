package clients

import (
	"context"
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"

	cerrors "github.com/contiamo/go-base/v3/pkg/errors"
	"github.com/contiamo/go-base/v3/pkg/tokens"
	"github.com/contiamo/go-base/v3/pkg/tracing"
	"github.com/opentracing/opentracing-go"
	otext "github.com/opentracing/opentracing-go/ext"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// APIError describes an error during unsuccessful HTTP request to an API
type APIError struct {
	// Status is the HTTP status of the API response
	Status int
	// Header is the set of response headers
	Header http.Header
	// Response is the response body
	Response []byte
}

// Error implements the error interface
func (e APIError) Error() string {
	return http.StatusText(e.Status)
}

// TokenProvider is a function that gets the token string for each request
type TokenProvider func() (token string, err error)

// TokenProviderFromCreator create a token provider out of token creator.
func TokenProviderFromCreator(tc tokens.Creator, reference string, opts tokens.Options) TokenProvider {
	return func() (token string, err error) {
		return tc.Create(reference, opts)
	}
}

var (
	// NoopTokenProvider is a token provider that returns an empty string which is ignored by `DoRequest`.
	NoopTokenProvider = TokenProvider(func() (token string, err error) { return "", nil })
)

// BaseAPIClient describes all basic HTTP client operations required to work with a JSON API
type BaseAPIClient interface {
	// GetBaseURL returns the base URL of the service which can be used in HTTP request tasks.
	GetBaseURL() string
	// DoRequest performs the HTTP request with the given parameters, marshals the payload and
	// unmarshals the response into the given output if the status code is successful
	DoRequest(ctx context.Context, method, path string, query url.Values, payload, out interface{}) error
}

// NewBaseAPIClient creates a new instance of the base API client implementation.
// Never use `debug=true` in production environments, it will leak sensitive data
func NewBaseAPIClient(basePath, tokenHeaderName string, tokenProvider TokenProvider, client *http.Client, debug bool) BaseAPIClient {
	return &baseAPIClient{
		Tracer:          tracing.NewTracer("clients", "BaseAPIClient"),
		basePath:        basePath,
		tokenHeaderName: tokenHeaderName,
		tokenProvider:   tokenProvider,
		client:          client,
		debug:           debug,
	}
}

type baseAPIClient struct {
	tracing.Tracer

	basePath        string
	tokenHeaderName string
	tokenProvider   TokenProvider
	client          *http.Client
	debug           bool
}

func (t baseAPIClient) GetBaseURL() string {
	return t.basePath
}

func (t baseAPIClient) DoRequest(ctx context.Context, method, path string, query url.Values, payload, out interface{}) (err error) {
	span, ctx := t.StartSpan(ctx, "DoRequest")
	defer func() {
		t.FinishSpan(span, err)
	}()
	span.SetTag("method", method)
	span.SetTag("path", path)

	queryString := query.Encode()
	span.SetTag("query", queryString)

	url := t.GetBaseURL() + path
	if queryString != "" {
		url += "?" + queryString
	}

	logrus := logrus.
		WithField("method", method).
		WithField("url", url)

	logrus.Debug("creating the request token...")
	token, err := t.tokenProvider()
	if err != nil {
		return errors.Wrap(err, "failed to create request token")
	}
	logrus.Debug("token created.")

	var payloadReader io.Reader
	if payload != nil {
		// streaming the payload
		r, w := io.Pipe()
		payloadReader = r
		encoder := json.NewEncoder(w)
		go func() {
			mErr := encoder.Encode(payload)
			if mErr != nil {
				_ = w.CloseWithError(mErr)
			} else {
				_ = w.Close()
			}
		}()
	}

	logrus.Debug("creating the HTTP request...")
	req, err := http.NewRequest(method, url, payloadReader)
	if err != nil {
		return errors.Wrap(err, "failed to create a new request")
	}

	// so, the HTTP request can be cancelled
	req = req.WithContext(ctx)

	req.Header.Add("Content-Type", "application/json")
	if token != "" {
		req.Header.Add(t.tokenHeaderName, token)
	} else {
		span.LogKV("token", "token value is empty, header was not set")
	}

	// set tracing headers so we can connect spans in different services
	err = opentracing.GlobalTracer().Inject(
		span.Context(),
		opentracing.HTTPHeaders,
		opentracing.HTTPHeadersCarrier(req.Header),
	)
	if err != nil {
		// this error should not crash the request, we log it and skip it
		otext.Error.Set(span, true)
		span.SetTag("tracing.inject.err", err.Error())
		logrus.Error(errors.Wrap(err, "cannot set tracing headers"))
		err = nil
	}
	logrus.Debug("HTTP request created.")

	logrus.Debug("doing request...")
	resp, err := t.client.Do(req)
	if err != nil {
		return errors.Wrap(err, "failed to do request")
	}
	logrus.Debug("request is done.")

	span.SetTag("response.status", resp.StatusCode)

	logrus.Debug("reading the response...")
	defer logrus.Debug("reading the response finished.")

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		contentType := resp.Header.Get("content-type")
		contentType = strings.ToLower(contentType)
		span.SetTag("resp.contentType", contentType)

		if strings.Contains(contentType, "json") {
			decoder := json.NewDecoder(resp.Body)
			return errors.Wrap(decoder.Decode(out), "failed to decode JSON response")
		}

		return nil
	}

	// these are the cases we can clearly map validation errors,
	// should effectively be server errors because they indicate some kind of bug in our implementation,
	// the Hub http layer validation should be strong enough to capture user fixable errors
	switch resp.StatusCode {
	case http.StatusUnauthorized:
		return cerrors.ErrAuthorization
	case http.StatusForbidden:
		return cerrors.ErrPermission
	case http.StatusNotFound:
		return cerrors.ErrNotFound
	case http.StatusNotImplemented:
		return cerrors.ErrNotImplemented
	default:
		if t.debug {
			// ignore the error on purpose here
			requestBody, _ := json.Marshal(payload)
			span.LogKV("request.body", string(requestBody))
		}

		// general error processing
		response, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return errors.Wrap(err, "failed to read response body")
		}
		span.LogKV("response.body", string(response))
		err = APIError{
			Status:   resp.StatusCode,
			Header:   resp.Header.Clone(),
			Response: response,
		}
		logrus.Error(errors.Wrap(err, "request failed"))
		logrus.Error(string(response))
		return err
	}
}
