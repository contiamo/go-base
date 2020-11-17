package migrations

import (
	"net/http"

	"github.com/contiamo/go-base/v2/pkg/fileutils/union"
)

// SQLAssets determines which filesystem object is used
// for the migrations flow and which for the views flow.
// The `000_init.sql` but live within the migrations
// file system.
type SQLAssets struct {
	Migrations http.FileSystem
	Views http.FileSystem
}

// NewSQLAssets creates a new filesystem that is compatible
// with the migration utilities.
func NewSQLAssets(assets SQLAssets) http.FileSystem {
	return union.New(map[string]http.FileSystem{
		"migrations":      assets.Migrations,
		"views": assets.Views,
	})
}
