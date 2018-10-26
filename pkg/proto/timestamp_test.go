package proto

import (
	"testing"
	"time"

	"github.com/golang/protobuf/ptypes"

	"github.com/golang/protobuf/ptypes/timestamp"
)

func Test_NullTimstamp_Scan(t *testing.T) {
	now := time.Now()
	var myTimestamp *timestamp.Timestamp
	emptyTimestamp := &timestamp.Timestamp{}
	nowTimestamp := ptypes.TimestampNow()

	scenarios := []struct {
		name  string
		value time.Time
		ts    *timestamp.Timestamp
	}{
		{"uninitilized timestamp", now, myTimestamp},
		{"empty timestamp", now, emptyTimestamp},
		{"non-empty timestamp", now, nowTimestamp},
	}

	for _, s := range scenarios {
		t.Run(s.name, func(t *testing.T) {
			scanner := NullTimestamp(&s.ts)

			err := scanner.Scan(s.value)
			if err != nil {
				t.Fatalf("unexpected scan error %s", err.Error())
			}

			if s.ts == nil {
				t.Fatal("expected scan to initialize the timestamp")
			}

			converted, err := ptypes.Timestamp(s.ts)
			if err != nil {
				t.Fatalf("unexpected error %s", err.Error())
			}

			if !now.Equal(converted) {
				t.Fatalf(
					"expected scan to set value to %s, got %s",
					now.Format(time.RFC3339),
					converted.Format(time.RFC3339),
				)
			}
		})
	}
}

func Test_NullTimstamp_ScanNil(t *testing.T) {
	var myTimestamp *timestamp.Timestamp

	scanner := NullTimestamp(&myTimestamp)

	err := scanner.Scan(nil)
	if err != nil {
		t.Fatalf("unexpected scan error %s", err.Error())
	}

	if myTimestamp != nil {
		t.Fatal("expected scan to not initialize the nil timestamp")
	}
}
