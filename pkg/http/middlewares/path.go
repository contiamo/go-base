package middleware

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi"
	"github.com/google/uuid"
)

// PathWithCleanID replace string values that look like ids (uuids and int) with "*"
func PathWithCleanID(r *http.Request) string {
	pathParts := strings.Split(r.URL.Path, "/")
	for i, part := range pathParts {
		if _, err := uuid.Parse(part); err == nil {
			pathParts[i] = "*"
			continue
		}
		if _, err := strconv.Atoi(part); err == nil {
			pathParts[i] = "*"
		}

	}
	return strings.Join(pathParts, "/")
}

// MethodAndPathCleanID replace string values that look like ids (uuids and int)
// with "*"
func MethodAndPathCleanID(r *http.Request) string {
	pathParts := strings.Split(r.URL.Path, "/")
	for i, part := range pathParts {
		if _, err := uuid.Parse(part); err == nil {
			pathParts[i] = "*"
			continue
		}
		if _, err := strconv.Atoi(part); err == nil {
			pathParts[i] = "*"
		}

	}
	return "HTTP " + r.Method + " " + strings.Join(pathParts, "/")
}

// ChiRouteName replace route parameters from the Chi route mux with "*"
func ChiRouteName(r *http.Request) string {
	return "HTTP " + r.Method + " " + chi.RouteContext(r.Context()).RoutePattern()
}
