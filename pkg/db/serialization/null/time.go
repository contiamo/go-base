package null

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"reflect"
	"time"

	"github.com/lib/pq"
)

// Time represents a time.Time that may be null. Time implements the
// sql.Scanner interface so it can be used as a scan destination, similar to
// sql.NullString.
type Time pq.NullTime

// TimeFrom returns a Time instance from a time.Time instance
func TimeFrom(input time.Time) Time {
	return Time{Time: input, Valid: true}
}

// Now returns the current local time as a null.Time
func Now() Time {
	return TimeFrom(time.Now())
}

// Format returns a textual representation of the time value formatted according to layout.  This
// is delegated to the time.Time object, see time.Time.Format for more details
func (nt Time) Format(layout string) string {
	if !nt.Valid {
		return "null"
	}
	return nt.Time.Format(layout)
}

// String implements the Stringer interface
func (nt Time) String() string {
	if !nt.Valid {
		return "null"
	}
	return nt.Time.Format("2006-01-02 15:04:05 -0700 MST")
}

// MarshalJSON marshalls Time to the primitive value `null` or the RFC3339
// string representation of the time value
func (nt Time) MarshalJSON() ([]byte, error) {
	if !nt.Valid {
		return []byte("null"), nil
	}
	val := fmt.Sprintf("\"%s\"", nt.Time.Format(time.RFC3339))
	return []byte(val), nil
}

// UnmarshalJSON implements json.Unmarshaler.
// It supports string and null input.
func (nt *Time) UnmarshalJSON(data []byte) error {
	var err error
	var v interface{}
	if err = json.Unmarshal(data, &v); err != nil {
		return err
	}
	switch x := v.(type) {
	case string:
		if x == "" || x == "null" {
			nt.Valid = false
			return nil
		}
		err = nt.Time.UnmarshalJSON(data)
	case nil:
		nt.Valid = false
		return nil
	default:
		err = fmt.Errorf("json: cannot unmarshal %v into Go value of type Time", reflect.TypeOf(v).Name())
	}
	nt.Valid = err == nil
	return err
}

// MarshalText implements the encoding.TextMarshaler interface.
func (nt Time) MarshalText() ([]byte, error) {
	if !nt.Valid {
		return []byte(`null`), nil
	}
	return nt.Time.MarshalText()
}

// UnmarshalText implements the encoding.TextUnmarshaler interface.
func (nt *Time) UnmarshalText(text []byte) error {
	str := string(text)
	if str == "" || str == "null" {
		nt.Valid = false
		return nil
	}
	if err := nt.Time.UnmarshalText(text); err != nil {
		return err
	}
	nt.Valid = true
	return nil
}

// Value implements the driver Valuer interface.
func (nt Time) Value() (driver.Value, error) {
	if !nt.Valid {
		return nil, nil
	}
	return nt.Time, nil
}

// Scan implements the sql.Scanner interface.
func (nt *Time) Scan(value interface{}) error {
	nt.Time, nt.Valid = value.(time.Time)
	return nil
}
