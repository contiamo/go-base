// This file is auto-generated, DO NOT EDIT.
//
// Source:
//     Title: Hub Service
//     Version: 0.1.0
package testpkg

// User service user object
type User struct {
	Gender *string `json:"gender,omitempty"`
	Name string `json:"name"`
	PreferredConnection ConnectionSpec `json:"preferredConnection,omitempty"`
}
