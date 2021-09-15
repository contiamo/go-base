package migrations

import (
	"context"
	"database/sql"
	"fmt"
	"io/ioutil"
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

	s, err := ioutil.ReadAll(file)
	if err != nil {
		return "", fmt.Errorf("getSQL failed: %w", err)
	}

	return string(s), nil
}

// Lock to ensure multiple migrations cannot occur simultaneously
const lockNum = int64(47831730) // arbitrary random number

func acquireAdvisoryLock(ctx context.Context, db *sql.DB) error {
	_, err := db.ExecContext(ctx, "select pg_advisory_lock($1)", lockNum)
	return err
}

func releaseAdvisoryLock(ctx context.Context, db *sql.DB) error {
	_, err := db.ExecContext(ctx, "select pg_advisory_unlock($1)", lockNum)
	return err
}
