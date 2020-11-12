package errors

// GeneralError represents a system error exposed to the user
type GeneralError struct {
	// Message of the validation error
	Message string `json:"message"`
	// Type of the error
	Type ErrorType `json:"type"`
}
