package errors

import (
	"errors"

	"github.com/contiamo/go-base/pkg/models"
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
	// ErrNotFound is the standard entiry not found error
	ErrNotFound = errors.New("The requested object was not found")
	// ErrNotImplemented is intended to be used when stubbing new endpoints
	ErrNotImplemented = errors.New("Method is not implemented")
)

// ValidationErrors contains errors organized by validated fields
// for now it's just an alias to the validation library we use
type ValidationErrors = validation.Errors

// ValidationErrorsToFieldErrorResponse converts validation errors to the format that is
// served by HTTP handlers
func ValidationErrorsToFieldErrorResponse(errs ValidationErrors) (fieldErrResp models.FieldErrorResponse) {
	fieldErrResp.Errors = make([]models.FieldError, 0, len(errs))
	for key, fieldErr := range errs {
		if fieldErr == nil {
			continue
		}
		fieldErrResp.Errors = append(fieldErrResp.Errors, models.FieldError{
			GeneralError: models.GeneralError{
				Type:    models.FieldErrorType,
				Message: fieldErr.Error(),
			},
			Key: key,
		})
	}
	return fieldErrResp
}
