package proto

import (
	"database/sql/driver"
	"fmt"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"
)

// NullProtoTimestamp wraps a timestamp.Timestamp into a Scanner/Valuer
// interface
type NullProtoTimestamp struct {
	Timestamp **timestamppb.Timestamp
}

// Scan implements the Scanner interface.
func (nt *NullProtoTimestamp) Scan(value interface{}) error {
	if value == nil {
		return nil
	}

	if *nt.Timestamp == nil {
		*nt.Timestamp = &timestamppb.Timestamp{}
	}

	switch v := value.(type) {
	case time.Time:
		ts := timestamppb.New(v)

		*nt.Timestamp = ts
		return ts.CheckValid()
	default:
		return fmt.Errorf("unknown value type")
	}
}

// Value implements the driver Valuer interface.
func (nt NullProtoTimestamp) Value() (driver.Value, error) {
	if nt.Timestamp == nil {
		return nil, nil
	}

	ts := (*nt.Timestamp)
	return ts.AsTime(), ts.CheckValid()
}

// NullTimestamp converts a timestamppb.Timestamp into a wrapped NullProtoTimestamp
func NullTimestamp(t **timestamppb.Timestamp) *NullProtoTimestamp {
	return &NullProtoTimestamp{
		Timestamp: t,
	}
}
