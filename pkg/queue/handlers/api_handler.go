package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/contiamo/go-base/v4/pkg/http/clients"
	"github.com/contiamo/go-base/v4/pkg/queue"
	"github.com/contiamo/go-base/v4/pkg/tracing"
	"github.com/sirupsen/logrus"
)

// NewJSONAPIHandler creates a task handler that makes an JSON HTTP request to a target
// API using the provided BaseAPIClient.
//
// The response from the request must be valid JSON or a stream of new line-separated
// JSON objects, otherwise the task will fail
//
// The BaseAPIClient is responsible for bringing its own TokenProvider.
//
// The NewAPIRequestHandler can be preferred if the request is not a JSON payload.
//
// The NewJSONAPIHandler can be preferred because it is easier to mock the BaseAPIClient
// for tests.
//
// Example usage:
//
// 		client := clients.NewBaseAPIClient(
// 			"", // use an empty baseURL because the task spec will hold the URL
// 			"X-Auth",
// 			clients.TokenProviderFromCreator(&creator, "apiRequestTask", tokens.Options{}),
// 			http.DefaultClient,
// 			false,
// 		)
// 		handler := NewJSONAPIHandler(client)
//
//
// Alternatively, use it within your custom task handler, this is required if the client
// behavior is dependent on the task spec:
//
// 		type customHandler struct {
// 			tracing.Tracer
// 		}
// 		func (h customHandler) Process(ctx context.Context, task queue.Task, heartbeats chan<- queue.Progress) (err error) {
// 			span, ctx := h.StartSpan(ctx, "Process")
// 			defer func() {
// 				close(heartbeats)
// 				heartbeats = nil
// 				h.FinishSpan(span, err)
// 			}()
//
// 			var spec tasks.CustomSpec
// 			err = json.Unmarshal(task.Spec, &spec)
// 			if err != nil {
// 				return err
// 			}
//
// 			creator := specSpecificTokenCreator{
// 				projectID: spec.ProjectID,
// 			}
// 			client := clients.NewBaseAPIClient(
// 				"", // use an empty baseURL because the task spec will hold the URL
// 				"Auth",
// 				clients.TokenProviderFromCreator(&creator, "apiRequestTask", tokens.Options{}),
// 				http.DefaultClient,
// 				false,
// 			)
//			client = clients.WithRetry(client, maxAttempts, backoff.Exponential())
// 			taskHandler := handlers.NewJSONAPIHandler(client)
//
// 			return taskHandler.Process(ctx, task, heartbeats)
// 		}
func NewJSONAPIHandler(client clients.BaseAPIClient) queue.TaskHandler {
	return jsonAPIHandler{
		Tracer: tracing.NewTracer("handlers", "JSONAPIHandler"),
		client: client,
	}
}

type jsonAPIHandler struct {
	tracing.Tracer
	client clients.BaseAPIClient
}

func (h jsonAPIHandler) Process(ctx context.Context, task queue.Task, heartbeats chan<- queue.Progress) (err error) {
	span, ctx := h.StartSpan(ctx, "Process")
	defer func() {
		close(heartbeats)
		h.FinishSpan(span, err)
	}()
	span.SetTag("task.id", task.ID)
	span.SetTag("task.queue", task.Queue)
	span.SetTag("task.type", task.Type)
	span.SetTag("task.spec", string(task.Spec))

	logrus := logrus.WithField("type", task.Type).WithField("id", task.ID)

	logrus.Debug("starting the API request task...")

	// clone the base client
	taskClient := h.client

	var progress APIRequestProgress
	defer func() {
		// we check for errSerializingHearbeat so we don't cause
		// a recursion call
		if err == nil || err == ErrSerializingHearbeat {
			return
		}
		message := err.Error()
		progress.ErrorMessage = &message
		_ = sendAPIRequestProgress(progress, heartbeats)
	}()

	var spec APIRequestTaskSpec
	err = json.Unmarshal(task.Spec, &spec)
	if err != nil {
		return err
	}

	progress.Stage = RequestPreparing
	err = sendAPIRequestProgress(progress, heartbeats)
	if err != nil {
		return err
	}

	headers := http.Header{}
	headers.Add("User-Agent", "Contiamo API Request Task")
	for name, value := range spec.RequestHeaders {
		headers.Add(name, value)
	}

	taskClient = taskClient.WithHeader(headers)
	if !spec.Authorized {
		// clear the token provider, which will then disable auth
		// otherwise we assume the api client is already configured
		// with the required auth provider
		taskClient = taskClient.WithTokenProvider(clients.NoopTokenProvider)
	}

	progress.Stage = RequestPending
	err = sendAPIRequestProgress(progress, heartbeats)
	if err != nil {
		return err
	}

	now := time.Now()
	defer func() {
		duration := time.Since(now)
		progress.Duration = &duration
		err := sendAPIRequestProgress(progress, heartbeats)
		if err != nil {
			logrus.Error(err)
		}
	}()

	var payload interface{} = nil
	if spec.RequestBody != "" {
		payload = spec.RequestBody
	}

	resp, err := taskClient.DoRequestWithResponse(ctx, spec.Method, spec.URL, nil, payload)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	contentType := resp.Header.Get("content-type")
	if !strings.Contains(contentType, "json") {
		return fmt.Errorf(
			"unexpected response content type, expected JSON, got `%s`",
			contentType,
		)
	}

	progress.Stage = RequestResponse
	progress.ReturnedStatus = &resp.StatusCode
	err = sendAPIRequestProgress(progress, heartbeats)
	if err != nil {
		return err
	}

	// the task would time out if the heartbeat was not sent for 30 seconds
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()
	go func() {
		for {
			select {
			case <-ticker.C:
				err := sendAPIRequestProgress(progress, heartbeats)
				if err != nil {
					logrus.Error(err)
				}
			case <-ctx.Done():
				return
			}
		}
	}()

	decoder := json.NewDecoder(resp.Body)
	for {
		err = ctx.Err()
		if err != nil {
			return err
		}
		var m json.RawMessage
		err = decoder.Decode(&m)
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		respString := string(m)
		progress.ReturnedBody = &respString
		err = sendAPIRequestProgress(progress, heartbeats)
		if err != nil {
			return err
		}
	}

	if spec.ExpectedStatus != resp.StatusCode {
		return fmt.Errorf("expected status %d but got %d", spec.ExpectedStatus, resp.StatusCode)
	}

	return nil
}
