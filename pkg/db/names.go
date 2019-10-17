package db

import (
	"strings"

	uuid "github.com/satori/go.uuid"
)

// GenerateSQLName generates a unique safe name that can be used for a database or table name
func GenerateSQLName() string {
	// a SQL identifier can't start with a number
	return "s" + strings.Replace(uuid.NewV4().String(), "-", "", -1)
}
