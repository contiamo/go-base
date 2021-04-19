package proto

import (
	"testing"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"
)

func Test_NullTimstamp_Scan(t *testing.T) {
	now := time.Now()
	var myTimestamp *timestamppb.Timestamp
	emptyTimestamp := &timestamppb.Timestamp{}
	nowTimestamp := timestamppb.Now()

	scenarios := []struct {
		name  string
		value time.Time
		ts    *timestamppb.Timestamp
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

			converted := s.ts.AsTime()
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
	var myTimestamp *timestamppb.Timestamp

	scanner := NullTimestamp(&myTimestamp)

	err := scanner.Scan(nil)
	if err != nil {
		t.Fatalf("unexpected scan error %s", err.Error())
	}

	if myTimestamp != nil {
		t.Fatal("expected scan to not initialize the nil timestamp")
	}
}
