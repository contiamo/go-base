package errors

import (
	"errors"
	"testing"

	"github.com/contiamo/go-base/pkg/models"
	"github.com/stretchr/testify/require"
)

func TestValidationErrorsToFieldErrorResponse(t *testing.T) {
	cases := []struct {
		name     string
		errs     ValidationErrors
		expected models.FieldErrorResponse
	}{
		{
			name:     "Returns empty response when errors are nil",
			expected: models.FieldErrorResponse{Errors: []models.FieldError{}},
		},
		{
			name:     "Returns empty response when errors are empty",
			errs:     ValidationErrors{},
			expected: models.FieldErrorResponse{Errors: []models.FieldError{}},
		},
		{
			name: "Returns errors in the response when errors are not empty",
			errs: ValidationErrors{
				"field1": errors.New("bad field1"),
				"field2": errors.New("bad field2"),
			},
			expected: models.FieldErrorResponse{
				Errors: []models.FieldError{
					models.FieldError{
						GeneralError: models.GeneralError{
							Type:    models.FieldErrorType,
							Message: "bad field1",
						},
						Key: "field1",
					},
					models.FieldError{
						GeneralError: models.GeneralError{
							Type:    models.FieldErrorType,
							Message: "bad field2",
						},
						Key: "field2",
					},
				},
			},
		},
		{
			name: "Does not include nil errors in the response",
			errs: ValidationErrors{
				"field1": errors.New("bad field1"),
				"field2": nil,
			},
			expected: models.FieldErrorResponse{
				Errors: []models.FieldError{
					models.FieldError{
						GeneralError: models.GeneralError{
							Type:    models.FieldErrorType,
							Message: "bad field1",
						},
						Key: "field1",
					},
				},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.expected, ValidationErrorsToFieldErrorResponse(tc.errs))
		})
	}
}
