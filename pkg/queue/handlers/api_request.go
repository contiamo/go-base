package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/contiamo/go-base/v4/pkg/queue"
	"github.com/contiamo/go-base/v4/pkg/tokens"
	"github.com/contiamo/go-base/v4/pkg/tracing"
	"github.com/opentracing/opentracing-go"
	otext "github.com/opentracing/opentracing-go/ext"
	"github.com/sirupsen/logrus"
)

var (
	// APIRequestTask marks a task as an API request task
	APIRequestTask queue.TaskType = "api-request"
)

// APIRequestTaskSpec describes the specification of the API request task
type APIRequestTaskSpec struct {
	// Method to use for the API request
	Method string `json:"method"`
	// URL is the target URL for the request.
	// Must be an absolute URL that contains the scheme and the host components.
	URL string `json:"url"`
	// RequestBody to send
	RequestBody string `json:"requestBody"`
	// RequestHeaders to send
	RequestHeaders map[string]string `json:"requestHeaders"`
	// Authorized if `true` the task will send a header with the
	// signed JWT token as a part of the request
	Authorized bool `json:"authorized"`
	// ExpectedStatus is an HTTP status expected as a response.
	// If it does not match the actual status the task fails
	ExpectedStatus int `json:"expectedStatus"`
}

type APIRequestStage string

var (
	// RequestPreparing means the task is preparing the request parameters and the body
	RequestPreparing APIRequestStage = "preparing"
	// RequestPending means the request was sent, awaiting the response
	RequestPending APIRequestStage = "pending"
	// RequestResponse means the response was received
	RequestResponse APIRequestStage = "response"
)

// APIRequestProgress describes the progress of the API request task stored during
// the heartbeat handling
type APIRequestProgress struct {
	// Stage is the current stage of the API request task
	Stage APIRequestStage `json:"stage,omitempty"`
	// Duration of the HTTP request
	Duration *time.Duration `json:"duration,omitempty"`
	// ReturnedStatus is a status returned from the target endpoint
	ReturnedStatus *int `json:"returnedStatus,omitempty"`
	// ReturnedBody is a body returned from the target endpoint
	ReturnedBody *string `json:"returnedBody,omitempty"`
	// ErrorMessage contains an error message string if it occurs during the update process
	ErrorMessage *string `json:"errorMessage,omitempty"`
}

// NewAPIRequestHandler creates a task handler that makes an HTTP request to a target API.
// The response from the request must be valid JSON or a stream of new line-separated
// JSON objects, otherwise the task will fail.
func NewAPIRequestHandler(tokenHeaderName string, tokenCreator tokens.Creator, client *http.Client) queue.TaskHandler {
	return &apiRequestHandler{
		Tracer:          tracing.NewTracer("handlers", "APIRequestHandler"),
		tokenHeaderName: tokenHeaderName,
		tokenCreator:    tokenCreator,
		client:          client,
	}
}

type apiRequestHandler struct {
	tracing.Tracer
	tokenHeaderName string
	tokenCreator    tokens.Creator
	client          *http.Client
}

func (h *apiRequestHandler) Process(ctx context.Context, task queue.Task, heartbeats chan<- queue.Progress) (err error) {
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

	var payload io.Reader
	if spec.RequestBody != "" {
		payload = strings.NewReader(spec.RequestBody)
	}

	req, err := http.NewRequest(spec.Method, spec.URL, payload)
	if err != nil {
		return err
	}

	req.Header.Add("User-Agent", "Contiamo API Request Task")

	for name, value := range spec.RequestHeaders {
		req.Header.Add(name, value)
	}

	if spec.Authorized {
		token, err := h.tokenCreator.Create("apiRequestTask", tokens.Options{})
		if err != nil {
			return err
		}

		req.Header.Add(h.tokenHeaderName, token)
	}

	err = opentracing.GlobalTracer().Inject(
		span.Context(),
		opentracing.HTTPHeaders,
		opentracing.HTTPHeadersCarrier(req.Header),
	)
	if err != nil {
		otext.Error.Set(span, true)
		span.SetTag("tracing.inject.err", err.Error())
		err = nil
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

	resp, err := h.client.Do(req.WithContext(ctx))
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

func sendAPIRequestProgress(progress APIRequestProgress, heartbeats chan<- queue.Progress) (err error) {
	logrus.
		WithField("method", "sendAPIRequestProgress").
		Debugf("%+v", progress)

	bytes, err := json.Marshal(progress)
	if err != nil {
		logrus.Error(err)
		return ErrSerializingHearbeat
	}

	heartbeats <- bytes
	return nil
}
