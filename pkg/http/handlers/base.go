package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"io"
	"net/http"

	cerrors "github.com/contiamo/go-base/v4/pkg/errors"
	"github.com/contiamo/go-base/v4/pkg/tracing"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/pkg/errors"
)

const (
	// Megabyte is a pre-defined maximum payload size that can be used in
	// NewBaseHandler
	Megabyte = 1024 * 1024
)

// ErrorParser is a function that parses an error into an HTTP status code and response body.
type ErrorParser = func(ctx context.Context, err error, debug bool) (int, interface{})

// BaseHandler contains all the base functions every handler should have
type BaseHandler interface {
	tracing.Tracer
	// Read reads the body and tries to decode JSON from it to the given output type
	Parse(r *http.Request, out interface{}) error
	// Write writes the response serving the given object as JSON with the given status
	Write(ctx context.Context, w http.ResponseWriter, status int, obj interface{})
	// Error serves the proper error object based on the given error and its type
	Error(context.Context, http.ResponseWriter, error)
}

// NewBaseHandler creates a new base HTTP handler that
// contains shared logic among all the handlers.
// The handler supports parsing and writing JSON objects
// `maxBodyBytes` is the maximal request body size, < 0 means the default Megabyte. Using 0 will disable the limit.
// `componentName` is used for tracing to identify to which
// component this handler belongs to.
func NewBaseHandler(componentName string, maxBodyBytes int64, debug bool) *Handler {
	if maxBodyBytes < 0 {
		maxBodyBytes = Megabyte
	}
	return &Handler{
		Tracer:       tracing.NewTracer("handlers", componentName),
		MaxBodyBytes: maxBodyBytes,
		Debug:        debug,
	}
}

// NewBasehandlerWithTracer create a new base HTTP handler, like NewBaseHandler, but allows
// the caller to configure the Tracer implementation independently.
//
// Deprecated: you can now configure/override the default Tracer using
//
//    h := NewBaseHandler(componentName, maxBodyBytes, debug)
//    h.Tracer = tracing.NewTracer("handlers", componentName)
func NewBaseHandlerWithTracer(tracer tracing.Tracer, maxBodyBytes int64, debug bool) *Handler {
	if maxBodyBytes < 0 {
		maxBodyBytes = Megabyte
	}
	return &Handler{
		Tracer:       tracer,
		MaxBodyBytes: maxBodyBytes,
		ErrorParser:  DefaultErrorParser,
		Debug:        debug,
	}
}

// Handler is the default implementation of BaseHandler and is suitable for use in
// most REST API implementations.
type Handler struct {
	tracing.Tracer
	// ErrorParser is used to parse error objects into HTTP status codes and response bodies.
	ErrorParser ErrorParser
	// MaxBodyBytes is the maximal request body size, < 0 means the default Megabyte.
	// Using 0 will disable the limit and allow parsing streams.
	MaxBodyBytes int64
	// Debug was used to enable/disable Debug mode, when enabled error messages will be included in responses.
	Debug bool
}

func (h *Handler) Error(ctx context.Context, w http.ResponseWriter, err error) {
	span, _ := h.StartSpan(ctx, "Error")
	defer h.FinishSpan(span, nil)

	parser := h.ErrorParser
	if parser == nil {
		parser = DefaultErrorParser
	}

	status, resp := parser(ctx, err, h.Debug)
	h.Write(ctx, w, status, resp)
}

func (h *Handler) Write(ctx context.Context, w http.ResponseWriter, status int, obj interface{}) {
	span, _ := h.StartSpan(ctx, "Write")
	var err error
	defer func() {
		h.FinishSpan(span, err)
	}()

	if obj == nil {
		w.WriteHeader(status)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if obj != nil {
		enc := json.NewEncoder(w)
		err = enc.Encode(obj)
	}
}

func (h *Handler) Parse(r *http.Request, out interface{}) error {
	var body io.Reader = r.Body
	if h.MaxBodyBytes > 0 {
		body = io.LimitReader(r.Body, h.MaxBodyBytes)
	}

	dec := json.NewDecoder(body)
	err := dec.Decode(out)
	if err != nil {
		return &parseError{cause: err}
	}
	return nil
}

// DefaultErrorParser provides a reasonable default error parser that can handle
// the various sentile errors in go-base as well as ozzo-validation errors.
func DefaultErrorParser(ctx context.Context, err error, debug bool) (int, interface{}) {
	// hey what are you doing here?
	if err == nil {
		return http.StatusOK, nil
	}

	genErrResp := cerrors.ErrorResponse{
		Errors: []cerrors.APIErrorMessenger{
			&cerrors.GeneralError{
				Type:    cerrors.GeneralErrorType,
				Message: err.Error(),
			}},
	}

	// Handle sentinel errors
	switch err {
	case cerrors.ErrPermission:
		return http.StatusForbidden, genErrResp
	case cerrors.ErrAuthorization:
		return http.StatusUnauthorized, genErrResp
	case cerrors.ErrInternal:
		return http.StatusInternalServerError, genErrResp
	case cerrors.ErrInvalidParameters:
		return http.StatusBadRequest, genErrResp
	case cerrors.ErrUnmarshalling, cerrors.ErrForm:
		return http.StatusUnprocessableEntity, genErrResp
	case sql.ErrNoRows, cerrors.ErrNotFound:
		return http.StatusNotFound, genErrResp
	case cerrors.ErrNotImplemented:
		return http.StatusNotImplemented, genErrResp
	case cerrors.ErrUnsupportedMediaType:
		return http.StatusUnsupportedMediaType, genErrResp
	}

	// Handle error types that wrap other errors
	var parseErr parseError
	if errors.As(err, &parseErr) {
		return http.StatusUnprocessableEntity, genErrResp
	}

	var userErr cerrors.UserError
	if errors.As(err, &userErr) {
		return http.StatusBadRequest, genErrResp
	}

	switch e := err.(type) {
	case cerrors.ValidationErrors:
		return http.StatusUnprocessableEntity, cerrors.ValidationErrorsToFieldErrorResponse(e)
	case validation.Errors:
		return http.StatusUnprocessableEntity, cerrors.ValidationErrorsToFieldErrorResponse(e)
	default:
		if !debug {
			for idx, e := range genErrResp.Errors {
				genErrResp.Errors[idx] = e.Scrubbed(cerrors.ErrInternal.Error())
			}
		}
		return http.StatusInternalServerError, genErrResp
	}
}

type parseError struct {
	cause error
}

func (e parseError) Error() string {
	return cerrors.ErrUnmarshalling.Error() + ": " + e.cause.Error()
}

func (e parseError) Unwrap() error {
	return e.cause
}

func (e parseError) As(target interface{}) bool {
	return errors.As(e.cause, &target)
}

func (e parseError) Is(target error) bool {
	return errors.Is(e.cause, target)
}
