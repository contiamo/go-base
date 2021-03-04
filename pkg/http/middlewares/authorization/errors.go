package authorization

import "github.com/pkg/errors"

// Validation error constants
var (
	ErrMissingSub   = errors.New("sub is required")
	ErrExpiration   = errors.New("invalid exp")
	ErrTooEarly     = errors.New("token is not valid yet")
	ErrTooSoon      = errors.New("token used before issued")
	ErrInvalidParty = errors.New("invalid authorized party")
)
