package null

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/satori/go.uuid"
)

// UUID represents a uuid.UUID that may be null. UUID implements the
// sql.Scanner interface so it can be used as a scan destination, similar to
// sql.NullString.  Addtionally, it implements the json.Marshaller, json.Unmarshaller,
// encoding.TextMarshaller, and encoding.TextUnmarshaller
type UUID uuid.NullUUID

// UUIDFromBytes returns UUID converted from raw byte slice input.
// It will return error if the slice isn't 16 bytes long.
func UUIDFromBytes(input []byte) (u UUID, err error) {
	err = u.UnmarshalBinary(input)
	return
}

// UUIDFromString returns UUID parsed from string input.
// Input is expected in a form accepted by UnmarshalText.
func UUIDFromString(input string) (u UUID, err error) {
	err = u.UnmarshalText([]byte(input))
	return
}

// UUIDFrom returns UUID from a uuid.UUID
func UUIDFrom(input uuid.UUID) UUID {
	return UUID{UUID: input, Valid: true}
}

// String implements the Stringer interface
func (nu UUID) String() string {
	return nu.UUID.String()
}

// MarshalJSON marshalls UUID to the primitive value `null` or the RFC3339
// string representation of the time value
func (nu UUID) MarshalJSON() ([]byte, error) {
	if !nu.Valid {
		return []byte("null"), nil
	}
	return []byte(`"` + nu.String() + `"`), nil
}

// UnmarshalJSON implements json.Unmarshaler.
// It supports string and null input.
func (nu *UUID) UnmarshalJSON(data []byte) error {
	var err error
	var v interface{}
	if err = json.Unmarshal(data, &v); err != nil {
		return err
	}
	switch x := v.(type) {
	case string:
		if x == "" || x == "null" {
			nu.Valid = false
			return nil
		}
		err = nu.UUID.UnmarshalText([]byte(x))
	case nil:
		nu.Valid = false
		return nil
	default:
		err = fmt.Errorf("json: cannot unmarshal %v into Go value of type UUID", reflect.TypeOf(v).Name())
	}
	nu.Valid = err == nil
	return err
}

// MarshalText implements the encoding.TextMarshaler interface.
func (nu UUID) MarshalText() ([]byte, error) {
	if !nu.Valid {
		return []byte("null"), nil
	}
	return nu.UUID.MarshalText()
}

// UnmarshalText implements the encoding.TextUnmarshaler interface.
func (nu *UUID) UnmarshalText(text []byte) error {
	str := string(text)
	if str == "" || str == "null" {
		nu.Valid = false
		return nil
	}
	if err := nu.UUID.UnmarshalText(text); err != nil {
		return err
	}
	nu.Valid = true
	return nil
}

// MarshalBinary implements the encoding.BinaryMarshaler interface.
func (nu UUID) MarshalBinary() ([]byte, error) {
	if !nu.Valid {
		return []byte("null"), nil
	}
	return nu.UUID.MarshalBinary()
}

// UnmarshalBinary implements the encoding.BinaryUnmarshaler interface.
// It will return error if the slice isn't 16 bytes long.
func (nu *UUID) UnmarshalBinary(data []byte) (err error) {
	str := string(data)
	if str == "" || str == "null" {
		nu.Valid = false
		return nil
	}
	if err := nu.UUID.UnmarshalBinary(data); err != nil {
		return err
	}
	nu.Valid = true
	return nil
}

// Value implements the driver.Valuer interface.
func (nu UUID) Value() (driver.Value, error) {
	if !nu.Valid {
		return nil, nil
	}
	// Delegate to UUID Value function
	return nu.UUID.Value()
}

// Scan implements the sql.Scanner interface.
func (nu *UUID) Scan(src interface{}) error {
	if src == nil {
		nu.UUID, nu.Valid = uuid.Nil, false
		return nil
	}

	// Delegate to UUID Scan function
	nu.Valid = true
	return nu.UUID.Scan(src)
}
