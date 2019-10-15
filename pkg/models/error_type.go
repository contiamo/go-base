package models

// ErrorType : The type of the error response
type ErrorType string

const (
	GENERAL_ERROR ErrorType = "GeneralError"
	FIELD_ERROR   ErrorType = "FieldError"
)
