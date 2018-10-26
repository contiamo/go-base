package sql

import (
	"bytes"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

const (
	// the use of `$$$&` is a little magic to trick GORM, while scrubbing to the inputs, it
	// does a replacement, for example https://github.com/jinzhu/gorm/blob/master/scope.go#L355,
	// and the `$$$$&` will be replaced with `?&` in the final query.
	jsonArrayMatchAllQuery = "%s::jsonb $$$& ?::text[]"
	// the use of `$$$|` is a little magic to trick GORM, while scrubbing to the inputs, it
	// does a replacement, for example https://github.com/jinzhu/gorm/blob/master/scope.go#L355,
	// and the `$$$|` will be replaced with `?|` in the final query.
	jsonArrayPartialMatchQuery = "%s::jsonb $$$| ?::text[]"
)

// CreateJSONStringArrayFilter returns the sql query and search value to find rots where the JSONStringArray matches
// the supplied values.  If `matchAll` is true, then the `JSONStringArray` must contain all of the
// supplied values.
func CreateJSONStringArrayFilter(fieldName string, q []string, matchAll bool) (query string, value string) {
	for i, tag := range q {
		q[i] = strings.ToLower(tag)
	}

	if matchAll {
		query = fmt.Sprintf(jsonArrayMatchAllQuery, fieldName)
	} else {
		query = fmt.Sprintf(jsonArrayPartialMatchQuery, fieldName)
	}

	value = `{` + strings.Join(q, ",") + `}`
	return query, value
}

// CreateJSONSTringArrayIndex returns a create index statement for the JSONStringArray field on the specified table
// with the given name.  This index is suitable for array element matching, e.g. a tags field.
func CreateJSONSTringArrayIndex(name, table, field string) (statement string) {
	statement = fmt.Sprintf("CREATE INDEX IF NOT EXISTS %s ON %s USING gin (%s);", name, table, field)
	return
}

// JSONStringArray is a []string that is compatible with the sql driver.
// Value() validates the json format in the source, and returns an error if
// the json is not valid.  Scan does no validation.
type JSONStringArray []string

// Value implements the Value interfance and provides the the database value in
// a type that the driver can handle, in paritcular as a string.
func (a JSONStringArray) Value() (driver.Value, error) {
	j, err := json.Marshal(a)
	return j, err
}

// Scan implements the Scanner interface that will scan the Postgres JSON payload
// into the JSONStringArray *a
func (a *JSONStringArray) Scan(src interface{}) error {
	var (
		source []byte
	)

	switch v := src.(type) {
	case []byte:
		source, _ = src.([]byte)
	case string:
		// t, ok := src.(string)
		source = []byte(src.(string))
	case nil:
		*a = nil
		return nil
	default:
		return fmt.Errorf("sql: cannot convert %T to JSONStringArray", v)
	}

	err := json.Unmarshal(source, a)
	if err != nil {
		return err
	}

	return nil
}

type JSONStringMap map[string]string

// Value implements the Value interfance and provides the the database value in
// a type that the driver can handle, in paritcular as a string.
func (a JSONStringMap) Value() (driver.Value, error) {
	j, err := json.Marshal(a)
	return j, err
}

// Scan implements the Scanner interface that will scan the Postgres JSON payload
// into the JSONStringMap *a
func (a *JSONStringMap) Scan(src interface{}) error {
	var (
		source []byte
	)

	switch v := src.(type) {
	case []byte:
		source, _ = src.([]byte)
	case string:
		// t, ok := src.(string)
		source = []byte(src.(string))
	case nil:
		*a = nil
		return nil
	default:
		return fmt.Errorf("sql: cannot convert %T to JSONStringMap", v)
	}

	err := json.Unmarshal(source, a)
	if err != nil {
		return err
	}

	return nil
}

// JSONMap describes a universal JSON structure with dynamic types for its fields
type JSONMap map[string]interface{}

// Value implements the Value interfance and provides the the database value in
// a type that the driver can handle, in paritcular as a string.
func (m JSONMap) Value() (driver.Value, error) {
	if m == nil {
		return []byte("null"), nil
	}
	j, err := json.Marshal(m)
	return j, err
}

// Scan implements the Scanner interface that will scan the Postgres JSON payload
// into the JSONMap.
func (m *JSONMap) Scan(src interface{}) error {
	source, ok := src.([]byte)
	if !ok {
		return errors.New("type assertion .([]byte) failed")
	}
	if bytes.Equal(source, []byte("null")) {
		*m = nil
		return nil
	}

	var i interface{}
	err := json.Unmarshal(source, &i)
	if err != nil {
		return err
	}

	*m, ok = i.(map[string]interface{})
	if !ok {
		return errors.New("type assertion .(map[string]interface{}) failed")
	}

	return nil
}
