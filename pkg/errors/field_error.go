package errors

// FieldError represents a validation error
type FieldError struct {
	// Key of the validated field
	Key string `json:"key"`
	GeneralError
}
