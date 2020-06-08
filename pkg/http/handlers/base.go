package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"io"
	"net/http"
	"strings"

	cerrors "github.com/contiamo/go-base/pkg/errors"
	"github.com/contiamo/go-base/pkg/tracing"
	"github.com/pkg/errors"
)

const (
	// Megabyte is a pre-defined maximum payload size that can be used in
	// NewBaseHandler
	Megabyte = 1024 * 1024
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
// The handler supports parsing and writing JSON objects
// `maxBodyBytes` is the maximal request body size, < 0 means the default Megabyte.
// `componentName` is used for tracing to identify to which
// component this handler belongs to.
func NewBaseHandler(componentName string, maxBodyBytes int64, debug bool) BaseHandler {
	if maxBodyBytes < 0 {
		maxBodyBytes = Megabyte
	}
	return &baseHandler{
		Tracer:       tracing.NewTracer("handlers", componentName),
		maxBodyBytes: maxBodyBytes,
		debug:        debug,
	}
}

// NewBasehandlerWithTracer create a new base HTTP handler, like NewBaseHandler, but allows
// the caller to configure the Tracer implementation independently.
func NewBaseHandlerWithTracer(tracer tracing.Tracer, maxBodyBytes int64, debug bool) BaseHandler {
	if maxBodyBytes < 0 {
		maxBodyBytes = Megabyte
	}
	return &baseHandler{
		Tracer:       tracer,
		maxBodyBytes: maxBodyBytes,
		debug:        debug,
	}
}

type baseHandler struct {
	tracing.Tracer
	maxBodyBytes int64
	debug        bool
}

func (h *baseHandler) Error(ctx context.Context, w http.ResponseWriter, err error) {
	span, _ := h.StartSpan(ctx, "Error")
	defer h.FinishSpan(span, nil)

	if err == nil {
		w.WriteHeader(http.StatusOK)
		return
	}

	genErrResp := cerrors.GeneralErrorResponse{
		Errors: []cerrors.GeneralError{{
			Type:    cerrors.GeneralErrorType,
			Message: err.Error(),
		}},
	}

	// we can extend this error list in the future if needed
	switch err {
	case cerrors.ErrNotImplemented:
		h.Write(ctx, w, http.StatusNotImplemented, genErrResp)
		return
	case cerrors.ErrAuthorization:
		h.Write(ctx, w, http.StatusUnauthorized, genErrResp)
		return
	case cerrors.ErrPermission:
		h.Write(ctx, w, http.StatusForbidden, genErrResp)
		return
	case cerrors.ErrForm:
		h.Write(ctx, w, http.StatusUnprocessableEntity, genErrResp)
		return
	case sql.ErrNoRows, cerrors.ErrNotFound:
		h.Write(ctx, w, http.StatusNotFound, genErrResp)
		return
	}

	if strings.HasPrefix(err.Error(), cerrors.ErrUnmarshalling.Error()) {
		h.Write(ctx, w, http.StatusUnprocessableEntity, genErrResp)
		return
	}

	validationErrs, ok := err.(cerrors.ValidationErrors)
	if ok {
		h.Write(
			ctx,
			w,
			http.StatusUnprocessableEntity,
			cerrors.ValidationErrorsToFieldErrorResponse(validationErrs),
		)
		return
	}

	if !h.debug {
		genErrResp.Errors[0].Message = cerrors.ErrInternal.Error()
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
	if obj != nil {
		enc := json.NewEncoder(w)
		err = enc.Encode(obj)
	}
}

func (h *baseHandler) Parse(r *http.Request, out interface{}) error {
	limitedBody := io.LimitReader(r.Body, h.maxBodyBytes)
	dec := json.NewDecoder(limitedBody)
	err := dec.Decode(out)
	if err != nil {
		return errors.Wrap(err, cerrors.ErrUnmarshalling.Error())
	}
	return nil
}
