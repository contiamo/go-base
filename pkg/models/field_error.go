package models

// FieldError represents a validation error
type FieldError struct {
	GeneralError
	// Key of the validated field
	Key string `json:"key"`
}
