package parameters

import (
	"net/http"
	"strings"

	"github.com/go-chi/chi"
)

// Resolver is able to resolve a parameter value by name
type Resolver interface {
	// Resolve takes the HTTP request and resolves a parameters value by the given name
	Resolve(*http.Request, string) string
}

// NewChiResolver returns a parameter resolves that uses chi-based resolver
func NewChiResolver() Resolver {
	return &chiResolver{}
}

type chiResolver struct{}

func (c *chiResolver) Resolve(r *http.Request, name string) (val string) {
	val = chi.URLParam(r, name)
	if val == "" {
		val = r.URL.Query().Get(name)
	}
	return strings.TrimSpace(val)
}
