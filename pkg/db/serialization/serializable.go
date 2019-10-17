package serialization

import (
	"database/sql"
	"database/sql/driver"
)

// Serializable is an interface to allow serialization and deserialization to a db
type Serializable interface {
	sql.Scanner
	driver.Valuer
	// GetData returns the original wrapped value
	GetData() interface{}
}
