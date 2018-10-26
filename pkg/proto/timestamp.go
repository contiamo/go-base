package proto

import (
	"database/sql/driver"
	"fmt"
	"time"

	"github.com/golang/protobuf/ptypes"

	"github.com/golang/protobuf/ptypes/timestamp"
)

// NullProtoTimestamp wraps a timestamp.Timestamp into a Scanner/Valuer
// interface
type NullProtoTimestamp struct {
	Timestamp **timestamp.Timestamp
}

// Scan implements the Scanner interface.
func (nt *NullProtoTimestamp) Scan(value interface{}) error {
	if value == nil {
		return nil
	}

	if *nt.Timestamp == nil {
		*nt.Timestamp = &timestamp.Timestamp{}
	}

	switch v := value.(type) {
	case time.Time:
		ts, err := ptypes.TimestampProto(v)
		if err != nil {
			return err
		}
		*nt.Timestamp = ts
		return nil
	default:
		return fmt.Errorf("unknown value type")
	}
}

// Value implements the driver Valuer interface.
func (nt NullProtoTimestamp) Value() (driver.Value, error) {
	return ptypes.Timestamp(*nt.Timestamp)
}

// NullTimestamp converts a timestamp.Timestamp into a wrapped NullProtoTimestamp
func NullTimestamp(t **timestamp.Timestamp) *NullProtoTimestamp {
	return &NullProtoTimestamp{
		Timestamp: t,
	}
}
