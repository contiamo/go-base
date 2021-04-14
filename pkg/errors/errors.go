package errors

import (
	"errors"
	"sort"

	validation "github.com/go-ozzo/ozzo-validation"
)

var (
	// ErrPermission is the standard "Permission Denied" error
	ErrPermission = errors.New("You don't have required permission to perform this action")
	// ErrAuthorization is the standard "Unauthorized" error
	ErrAuthorization = errors.New("User is unauthorized, make sure you've logged in")
	// ErrInternal is the standard "Internal Server" error
	ErrInternal = errors.New("Internal server error, please try again later")
	// ErrInvalidParameters is the standard "Bad Request" error
	ErrInvalidParameters = errors.New("Some of the request parameters are not correct")
	// ErrUnmarshalling is the JSON deserialization error
	ErrUnmarshalling = errors.New("Failed to read JSON from the request body")
	// ErrForm is the form parsing error
	ErrForm = errors.New("Failed to parse the submitted form")
	// ErrNotFound is the standard entity not found error
	ErrNotFound = errors.New("The requested object was not found")
	// ErrNotImplemented is intended to be used when stubbing new endpoints
	ErrNotImplemented = errors.New("Method is not implemented")
)

// ValidationErrors contains errors organized by validated fields
// for now it's just an alias to the validation library we use
type ValidationErrors = validation.Errors

// ValidationErrorsToFieldErrorResponse converts validation errors to the format that is
// served by HTTP handlers
func ValidationErrorsToFieldErrorResponse(errs ValidationErrors) (resp ErrorResponse) {
	resp.Errors = make([]APIErrorMessenger, 0, len(errs))
	for key, fieldErr := range errs {
		if fieldErr == nil {
			continue
		}

		var e APIErrorMessenger
		e = GeneralError{
			Type:    GeneralErrorType,
			Message: fieldErr.Error(),
		}

		if key != "" {
			e = FieldError{
				GeneralError: GeneralError{
					Type:    FieldErrorType,
					Message: fieldErr.Error(),
				},
				Key: key,
			}
		}
		resp.Errors = append(resp.Errors, e)
	}

	// to always have deterministic results
	sort.Slice(resp.Errors, func(i, j int) bool {
		return resp.Errors[i].GetMessage() < resp.Errors[j].GetMessage()
	})
	return resp
}

// UserError is an error wrapper that represents a GeneralError caused by some
// user data or request. The  error message is considered safe to show to end users.
// The HTTP handler can recognize this error type and automatically parse it into a 400
// error code.
type UserError struct {
	cause error
}

// Error implements the error interface
func (e UserError) Error() string {
	return e.cause.Error()
}

// Unwrap implements the error wrapping interface to expose the source error
func (e UserError) Unwrap() error {
	return e.cause
}

// AsUserError wraps the error as a UserError type, this allows automatic
// handling by the BaseHandler
func AsUserError(err error) error {
	return UserError{cause: err}
}
