package clients

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"

	"github.com/cenkalti/backoff/v4"
	"github.com/contiamo/go-base/v4/pkg/tracing"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// WithRetry returns an implementation of BaseAPIClient that also adds logic to automatically retry
// requests on specific error cases.
func WithRetry(client BaseAPIClient, maxAttempts uint64, plan backoff.BackOff) ClientWithRetry {
	return ClientWithRetry{
		Tracer:      tracing.NewTracer("clients", "WithRetry"),
		client:      client,
		MaxAttempts: maxAttempts,
		Retryable:   IsRetryable,
		Plan:        plan,
	}
}

// ClientWithRetry is an implementation of BaseAPIClient that also adds logic to automatically retry
// requests on specific error cases.
type ClientWithRetry struct {
	tracing.Tracer
	client BaseAPIClient
	// Retryable is a function that tests if the response can be retried.
	// The default implementation is
	Retryable func(*http.Response, error) bool
	// MaxAttempts determines the maximum request attempts that will be made.
	MaxAttempts uint64
	// Plan is the base backoff plan that will be used, it will be wrapped with
	// context and max attempts plans. In general, you will set this to
	// either backoff ExponentialBackoff or ConstantBackoff.
	// The default plan is backoff.NewExponentialBackoff()
	Plan backoff.BackOff
}

// GetBaseURL implements BaseAPIClient
func (c ClientWithRetry) GetBaseURL() string {
	return c.client.GetBaseURL()
}

// WithHeader implements BaseAPIClient
func (c ClientWithRetry) WithHeader(headers http.Header) BaseAPIClient {
	return ClientWithRetry{
		Tracer:      c.Tracer,
		client:      c.client.WithHeader(headers),
		Retryable:   c.Retryable,
		MaxAttempts: c.MaxAttempts,
		Plan:        c.Plan,
	}
}

// WithTokenProvider implements BaseAPIClient
func (c ClientWithRetry) WithTokenProvider(tokenProvider TokenProvider) BaseAPIClient {
	return ClientWithRetry{
		Tracer:      c.Tracer,
		client:      c.client.WithTokenProvider(tokenProvider),
		Retryable:   c.Retryable,
		MaxAttempts: c.MaxAttempts,
		Plan:        c.Plan,
	}
}

// DoRequest implements BaseAPIClient
func (c ClientWithRetry) DoRequest(ctx context.Context, method, path string, query url.Values, payload, out interface{}) (err error) {
	span, ctx := c.StartSpan(ctx, "DoRequest")
	defer func() {
		c.FinishSpan(span, err)
	}()

	// non-2** status codes will be errors already
	resp, err := c.DoRequestWithResponse(ctx, method, path, query, payload)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	contentType := resp.Header.Get("content-type")
	contentType = strings.ToLower(contentType)
	span.SetTag("response_has_output_destination", out != nil)
	span.SetTag("resp.contentType", contentType)

	// handle the success response and parse the json payload into `out`
	if out != nil && strings.Contains(contentType, "json") {
		decoder := json.NewDecoder(resp.Body)
		return errors.Wrap(decoder.Decode(out), "failed to decode JSON response")
	}

	return nil
}

// DoRequestWithResponse implements BaseAPIClient
func (c ClientWithRetry) DoRequestWithResponse(ctx context.Context, method, path string, query url.Values, payload interface{}) (body *http.Response, err error) {
	span, ctx := c.StartSpan(ctx, "DoRequestWithResponse")
	defer func() {
		c.FinishSpan(span, err)
	}()

	span.SetTag("maxAttempts", c.MaxAttempts)

	var lastResp *http.Response

	basePlan := c.Plan
	if basePlan == nil {
		basePlan = backoff.NewExponentialBackOff()
	}

	// let the backoff plan handle the max case and context cancel
	// WithMaxRetries is not thread-safe, so we initialize it here
	plan := backoff.WithMaxRetries(
		backoff.WithContext(basePlan, ctx),
		// the attempts counter is 0 based
		c.MaxAttempts-1,
	)

	retryable := c.Retryable
	if c.Retryable == nil {
		retryable = IsRetryable
	}

	logger := logrus.WithContext(ctx).
		WithField("maxAttempts", c.MaxAttempts)

	var attempt int
	// note about the Retry error
	// 1. Retry will unwrap the Permanent error for us
	// 2. when we hit max retries, we will get the last operation error from the Retry
	// 3. will return the context error, if it is not nil
	err = backoff.Retry(func() error {
		var attemptErr error
		attempt++

		// we assume that BaseAPIClient implements Tracer, so we don't create
		// a subspan for each attempt, the DoRequestWithResponse will do that already

		// nolint:bodyclose // caller is now responsible for closing, if there is no error
		lastResp, attemptErr = c.client.DoRequestWithResponse(ctx, method, path, query, payload)
		if !retryable(lastResp, attemptErr) {
			logger.WithField("attempt", attempt).WithError(attemptErr).Error("permanent error")
			return backoff.Permanent(attemptErr)
		}

		logger.WithField("attempt", attempt).WithError(attemptErr).Debug("retryable error")
		return attemptErr
	}, plan)

	return lastResp, err
}

// IsRetryable is the default test to check if the client should retry a request.
func IsRetryable(_ *http.Response, err error) bool {
	if err == nil {
		return false
	}
	apiErr, ok := err.(APIError)
	if ok {
		switch apiErr.Status {
		case
			http.StatusBadRequest,
			http.StatusRequestTimeout,
			444, // connection closed without response,
			499, // client close request
			http.StatusInternalServerError,
			http.StatusBadGateway,
			http.StatusServiceUnavailable,
			http.StatusGatewayTimeout,
			599: // Network Connect Timeout Error:
			return true
		default:
			return false
		}
	}

	return false
}
