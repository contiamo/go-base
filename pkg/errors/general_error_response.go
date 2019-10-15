package errors

// GeneralErrorResponse - General error response that usually has a very generic message
type GeneralErrorResponse struct {
	// Errors is a list of errors
	Errors []GeneralError `json:"errors"`
}
