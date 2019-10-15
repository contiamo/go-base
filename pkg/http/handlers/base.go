package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"io"
	"net/http"

	"github.com/contiamo/go-base/pkg/errors"
	"github.com/contiamo/go-base/pkg/tracing"
)

const (
	maxRequestBodySize = 1024 * 1024 // 1MB
)

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
// `componentName` is used for tracing to identify to which
// component this handler belongs to.
func NewBaseHandler(componentName string, debug bool) BaseHandler {
	return &baseHandler{
		Tracer: tracing.NewTracer("handlers", componentName),
		debug:  debug,
	}
}

type baseHandler struct {
	tracing.Tracer
	debug bool
}

func (h *baseHandler) Error(ctx context.Context, w http.ResponseWriter, err error) {
	span, _ := h.StartSpan(ctx, "Error")
	defer h.FinishSpan(span, nil)

	genErrResp := errors.GeneralErrorResponse{
		Errors: []errors.GeneralError{{
			Type:    errors.GeneralErrorType,
			Message: err.Error(),
		}},
	}

	// we can extend this error list in the future if needed
	switch err {
	case errors.ErrNotImplemented:
		h.Write(ctx, w, http.StatusNotImplemented, genErrResp)
		return
	case errors.ErrAuthorization:
		h.Write(ctx, w, http.StatusUnauthorized, genErrResp)
		return
	case errors.ErrPermission:
		h.Write(ctx, w, http.StatusForbidden, genErrResp)
		return
	case errors.ErrUnmarshalling, errors.ErrForm:
		h.Write(ctx, w, http.StatusUnprocessableEntity, genErrResp)
		return
	case sql.ErrNoRows, errors.ErrNotFound:
		h.Write(ctx, w, http.StatusNotFound, genErrResp)
		return
	}

	validationErrs, ok := err.(errors.ValidationErrors)
	if ok {
		h.Write(
			ctx,
			w,
			http.StatusUnprocessableEntity,
			errors.ValidationErrorsToFieldErrorResponse(validationErrs),
		)
		return
	}

	if !h.debug {
		genErrResp.Errors[0].Message = errors.ErrInternal.Error()
	}

	h.Write(ctx, w, http.StatusInternalServerError, genErrResp)
}

func (h *baseHandler) Write(ctx context.Context, w http.ResponseWriter, status int, obj interface{}) {
	span, _ := h.StartSpan(ctx, "Write")
	var err error
	defer func() {
		h.FinishSpan(span, err)
	}()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	enc := json.NewEncoder(w)
	err = enc.Encode(obj)
}

func (h *baseHandler) Parse(r *http.Request, out interface{}) error {
	limitedBody := io.LimitReader(r.Body, maxRequestBodySize)
	dec := json.NewDecoder(limitedBody)
	err := dec.Decode(out)
	if err != nil {
		return errors.ErrUnmarshalling
	}
	return nil
}
