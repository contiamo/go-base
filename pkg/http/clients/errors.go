package clients

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	cerrors "github.com/contiamo/go-base/v4/pkg/errors"
)

type errorResponse struct {
	Errors []respError `json:"errors"`
}
type respError struct {
	Key     string `json:"key"`
	Message string `json:"message"`
}

// APIError describes an error during unsuccessful HTTP request to an API
type APIError struct {
	// Status is the HTTP status of the API response
	Status int
	// Header is the set of response headers
	Header http.Header
	// Response is the bytes of the response body
	Response []byte
}

// Error implements the error interface.
// Builds a complete error message out of all the errors served in the API response.
// This should be used in logging, tracing, debugging, etc.
// For user facing errors use `ValidationErrors` function.
func (e APIError) Error() (message string) {
	errs := e.ResponseErrors()

	messages := make([]string, 0, len(errs)+1) // plus the status message in the beginning

	status := http.StatusText(e.Status)
	if status == "" {
		status = strconv.Itoa(e.Status)
	}
	messages = append(messages, status)

	for _, err := range errs {
		messages = append(messages, err.Error())
	}

	return strings.Join(messages, "; ")
}

// ValidationErrors returns a map of validation errors in case they were present in response errors.
// Otherwise, returns `nil` if there were no validation errors in the response.
func (e APIError) ValidationErrors() (errs cerrors.ValidationErrors) {
	allErrs := e.ResponseErrors()

	// it's always not more than one `cerrors.ValidationErrors` on the list
	for _, err := range allErrs {
		switch typed := err.(type) {
		case cerrors.ValidationErrors:
			return typed
		}
	}
	return nil
}

// ResponseErrors returns a slice of all errors that  were present in the API response.
// All the field errors are folded into a single validation error map.
// All the general errors are mapped to regular errors.
// Returns `nil` if the response contained no errors in the expected JSON format.
// For the 522 status code it handles a special case where all the general errors become validation
// errors for the `connection` field.
func (e APIError) ResponseErrors() (errs []error) {
	var response errorResponse
	err := json.Unmarshal(e.Response, &response)
	if err != nil {
		return nil
	}

	validationErrs := make(cerrors.ValidationErrors, len(response.Errors))
	connectivityGeneralErrs := make([]string, 0, len(response.Errors))

	for _, respErr := range response.Errors {
		if respErr.Key != "" {
			validationErrs[respErr.Key] = errors.New(respErr.Message)
			continue
		}

		// special case for connectivity errors that should be turned into validation errors
		if e.Status == 522 {
			connectivityGeneralErrs = append(connectivityGeneralErrs, respErr.Message)
			continue
		}

		errs = append(errs, errors.New(respErr.Message))
	}

	if len(connectivityGeneralErrs) > 0 {
		if validationErrs["connection"] != nil {
			connectivityGeneralErrs = append(connectivityGeneralErrs, validationErrs["connection"].Error())
		}
		validationErrs["connection"] = errors.New(strings.Join(connectivityGeneralErrs, ". "))
	}

	if len(validationErrs) > 0 {
		errs = append(errs, validationErrs)
	}

	return errs
}
