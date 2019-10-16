package authorization

import (
	"encoding/json"
	"math"
	"strconv"
	"strings"
	"time"
)

// Timestamp provides a timestamp value that can handle JSON strings
// and numeric values
type Timestamp struct {
	time time.Time
}

// Time returns the embedded go time value
func (t Timestamp) Time() time.Time {
	return t.time
}

// FromTime creates a timestamp from an existing time value
func FromTime(t time.Time) Timestamp {
	return Timestamp{time: t}
}

// MarshalJSON implements the JSON marshal interface, returning
//  t as a Unix time, the number of seconds elapsed since
// January 1, 1970 UTC.
func (t Timestamp) MarshalJSON() ([]byte, error) {
	return []byte(strconv.FormatInt(t.time.UTC().Unix(), 10)), nil
}

// UnmarshalJSON implements json.Unmarshaler.
// It supports string and null input.
func (t *Timestamp) UnmarshalJSON(data []byte) error {
	var err error
	var numeric json.Number

	err = json.Unmarshal(data, &numeric)
	if err != nil {
		return err
	}

	timestampStr := numeric.String()
	switch {
	case strings.Contains(timestampStr, "T"):
		t.time, err = time.Parse(time.RFC3339, timestampStr)
	case strings.Contains(timestampStr, "."):
		var v float64
		v, err = numeric.Float64()
		t.time = timeFromFloat64(v)
	default:
		var v int64
		v, err = numeric.Int64()
		if err == nil {
			t.time = time.Unix(v, 0).UTC()
		}
	}
	return err
}

func timeFromFloat64(value float64) time.Time {
	secs, nsecs := math.Modf(value)
	return time.Unix(int64(secs), int64(nsecs*1e9)).UTC()
}
