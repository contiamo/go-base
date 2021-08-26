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
)

func WithRetry(client BaseAPIClient, maxAttempts uint64, plan backoff.BackOff) BaseAPIClient {
	backoff.WithMaxRetries(plan, maxAttempts)
	return clientWithRetry{
		Tracer:        tracing.NewTracer("clients", "WithRetry"),
		BaseAPIClient: client,
		maxAttempts:   maxAttempts,
		plan:          plan,
	}
}

type clientWithRetry struct {
	tracing.Tracer
	BaseAPIClient
	maxAttempts uint64
	plan        backoff.BackOff
}

func (c clientWithRetry) DoRequest(ctx context.Context, method, path string, query url.Values, payload, out interface{}) (err error) {
	span, ctx := c.StartSpan(ctx, "DoRequestWithRetry")
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

	if out != nil && strings.Contains(contentType, "json") {
		decoder := json.NewDecoder(resp.Body)
		return errors.Wrap(decoder.Decode(out), "failed to decode JSON response")
	}

	return nil
}

func (c clientWithRetry) DoRequestWithResponse(ctx context.Context, method, path string, query url.Values, payload interface{}) (body *http.Response, err error) {
	span, ctx := c.StartSpan(ctx, "DoRequestWithResponseWithRetry")
	defer func() {
		c.FinishSpan(span, err)
	}()

	span.SetTag("maxAttempts", c.maxAttempts)

	var lastResp *http.Response

	// let the backoff plan handle the max case and context cancel
	plan := backoff.WithMaxRetries(
		backoff.WithContext(c.plan, ctx),
		// the attempts counter is 0 based
		c.maxAttempts-1,
	)

	// note about the Retry error
	// 1. Retry will unwrap the Permanent error for us
	// 2. when we hit max retries, we will get the last operation error from the Retry
	// 3. will return the context error, if it is not nil
	err = backoff.Retry(func() error {
		var attemptErr error

		// nolint:bodyclose // caller is now responsible for closing, if there is no error
		lastResp, attemptErr = c.BaseAPIClient.DoRequestWithResponse(ctx, method, path, query, payload)
		if !isRetryable(attemptErr) {
			return backoff.Permanent(attemptErr)
		}

		return attemptErr
	}, plan)

	return lastResp, err
}

func isRetryable(err error) bool {
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
