package errors

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidationErrorsToErrorResponse(t *testing.T) {
	cases := []struct {
		name     string
		errs     ValidationErrors
		expected ErrorResponse
	}{
		{
			name:     "Returns empty response when errors are nil",
			expected: ErrorResponse{Errors: []APIErrorMessenger{}},
		},
		{
			name:     "Returns empty response when errors are empty",
			errs:     ValidationErrors{},
			expected: ErrorResponse{Errors: []APIErrorMessenger{}},
		},
		{
			name: "Returns errors in the response when errors are not empty",
			errs: ValidationErrors{
				"field1": errors.New("bad field1"),
				"field2": errors.New("bad field2"),
			},
			expected: ErrorResponse{
				Errors: []APIErrorMessenger{
					FieldError{
						GeneralError: GeneralError{
							Type:    FieldErrorType,
							Message: "bad field1",
						},
						Key: "field1",
					},
					FieldError{
						GeneralError: GeneralError{
							Type:    FieldErrorType,
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
			expected: ErrorResponse{
				Errors: []APIErrorMessenger{
					FieldError{
						GeneralError: GeneralError{
							Type:    FieldErrorType,
							Message: "bad field1",
						},
						Key: "field1",
					},
				},
			},
		},
		{
			name: "Does sort by key",
			errs: ValidationErrors{
				"b": errors.New("bad field b"),
				"a": errors.New("bad field a"),
			},
			expected: ErrorResponse{
				Errors: []APIErrorMessenger{
					FieldError{
						GeneralError: GeneralError{
							Type:    FieldErrorType,
							Message: "bad field a",
						},
						Key: "a",
					},
					FieldError{
						GeneralError: GeneralError{
							Type:    FieldErrorType,
							Message: "bad field b",
						},
						Key: "b",
					},
				},
			},
		},
		{
			name: "empty string keys are mapped to a GeneralError type",
			errs: ValidationErrors{
				"field1": errors.New("bad field1"),
				"":       errors.New("other generic validation"),
			},
			expected: ErrorResponse{
				Errors: []APIErrorMessenger{
					FieldError{
						GeneralError: GeneralError{
							Type:    FieldErrorType,
							Message: "bad field1",
						},
						Key: "field1",
					},
					GeneralError{
						Type:    GeneralErrorType,
						Message: "other generic validation",
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
