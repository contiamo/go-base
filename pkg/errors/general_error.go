package errors

// GeneralError represents a system error exposed to the user
type GeneralError struct {
	// Type of the error
	Type ErrorType `json:"type"`
	// Message of the validation error
	Message string `json:"message"`
}
