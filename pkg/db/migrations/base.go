package migrations

import (
	"fmt"
	"io"
	"net/http"
	"path/filepath"
)

// sqlKind is is used to distinguish where the SQL file opener will look for the SQL
type sqlKind string

const (
	migrations sqlKind = "migrations"
	views      sqlKind = "views"
)

func getSQL(name string, kind sqlKind, assets http.FileSystem) (string, error) {
	file, err := assets.Open(filepath.Join("/", string(kind), name))
	if err != nil {
		return "", fmt.Errorf("getSQL failed: %w", err)
	}

	s, err := io.ReadAll(file)
	if err != nil {
		return "", fmt.Errorf("getSQL failed: %w", err)
	}

	return string(s), nil
}
