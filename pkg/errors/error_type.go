package errors

// ErrorType : The type of the error response
type ErrorType string

const (
	// GeneralErrorType is a generic error type
	GeneralErrorType ErrorType = "GeneralError"
	// FieldErrorType is a field validation error type
	FieldErrorType ErrorType = "FieldError"
)
