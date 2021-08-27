package clients

import (
	"net/http"
)

// IsRetryable is the default test to check if the client should retry a request.
func IsRetryable(_ *http.Response, err error) bool {
	if err == nil {
		return false
	}
	apiErr, ok := err.(APIError)
	if ok {
		switch apiErr.Status {
		case
			http.StatusBadRequest,
			http.StatusRequestTimeout,
			444, // connection closed without response,
			499, // client close request
			http.StatusInternalServerError,
			http.StatusBadGateway,
			http.StatusServiceUnavailable,
			http.StatusGatewayTimeout,
			599: // Network Connect Timeout Error:
			return true
		default:
			return false
		}
	}

	return false
}
