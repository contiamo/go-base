package errors

// FieldErrorResponse - Error message that contains detailed information about certain parameters being incorrect
//
// Deprecated: replaced with the more generic ErrorResponse
type FieldErrorResponse struct {
	// Errors is a list of errors
	Errors []FieldError `json:"errors"`
}
