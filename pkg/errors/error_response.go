package errors

// ErrorType : The type of the error response
type ErrorType string

const (
	// GeneralErrorType is a generic error type
	GeneralErrorType ErrorType = "GeneralError"
	// FieldErrorType is a field validation error type
	FieldErrorType ErrorType = "FieldError"
)

// ErrorResponse is a generic API error response body
type ErrorResponse struct {
	// Errors is a list of errors
	Errors []APIErrorMessenger `json:"errors"`
}

// APIErrorMessenger represents an error message
type APIErrorMessenger interface {
	GetType() ErrorType
	GetMessage() string
	// Scrubbed returns a copy of the instance with the message
	// replaced the cleaned value
	Scrubbed(string) APIErrorMessenger
}

// FieldError represents a validation error
type FieldError struct {
	GeneralError
	// Key of the validated field
	Key string `json:"key"`
}

func (e FieldError) GetType() ErrorType {
	return e.Type
}

func (e FieldError) GetMessage() string {
	return e.Message
}

// Scrubbed implemented the APIErrorMessanger, the scrubbing
// is a noop because FieldError is safe for users by default.
func (e FieldError) Scrubbed(_ string) APIErrorMessenger {
	return e
}

// GeneralError represents a system error exposed to the user
type GeneralError struct {
	// Type of the error
	Type ErrorType `json:"type"`
	// Message of the validation error
	Message string `json:"message"`
}

func (e GeneralError) GetType() ErrorType {
	return e.Type
}

func (e GeneralError) GetMessage() string {
	return e.Message
}

// Scrubbed implemented the APIErrorMessanger, the scrubbing
// is replaces the message with the given message value.
func (e GeneralError) Scrubbed(message string) APIErrorMessenger {
	return GeneralError{
		Type:    e.Type,
		Message: message,
	}
}
