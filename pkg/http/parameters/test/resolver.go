package test

import (
	"net/http"

	"github.com/contiamo/go-base/v3/pkg/http/parameters"
)

// NewMockResolver creates a new mock resolver with the predefined value
func NewMockResolver(values map[string]string) parameters.Resolver {
	return &mockResolver{values}
}

type mockResolver struct {
	values map[string]string
}

func (r *mockResolver) Resolve(req *http.Request, name string) string {
	return r.values[name]
}
