package errors

// GeneralErrorResponse - General error response that usually has a very generic message
//
// Deprecated: replaced with the more generic ErrorResponse
type GeneralErrorResponse struct {
	// Errors is a list of errors
	Errors []GeneralError `json:"errors"`
}
