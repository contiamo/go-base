// This file is auto-generated, DO NOT EDIT.
//
// Source:
//     Title: Hub Service
//     Version: 0.1.0
package testpkg

import (
	validation "github.com/go-ozzo/ozzo-validation"
)

// HighlightIndicatorStart is an enum.
type HighlightIndicatorStart string

var (
	HighlightIndicatorStartValue0 HighlightIndicatorStart = "{{{"

	// KnownHighlightIndicatorStart is the list of valid HighlightIndicatorStart
	KnownHighlightIndicatorStart = []HighlightIndicatorStart{
		HighlightIndicatorStartValue0,
	}
	// KnownHighlightIndicatorStartString is the list of valid HighlightIndicatorStart as string
	KnownHighlightIndicatorStartString = []string{
		string(HighlightIndicatorStartValue0),
	}

	// InKnownHighlightIndicatorStart is an ozzo-validator for HighlightIndicatorStart
	InKnownHighlightIndicatorStart = validation.In(
		HighlightIndicatorStartValue0,
	)
)
