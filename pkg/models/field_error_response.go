package models

// FieldErrorResponse - Error message that contains detailed information about certain parameters being incorrect
type FieldErrorResponse struct {
	// Errors is a list of errors
	Errors []FieldError `json:"errors"`
}
