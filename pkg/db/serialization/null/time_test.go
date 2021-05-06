package null

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

var (
	timeString   = "2012-12-21T21:21:21Z"
	timeGoString = "2012-12-21 21:21:21 +0000 UTC"
	timeJSON     = []byte(`"` + timeString + `"`)
	timeValue, _ = time.Parse(time.RFC3339, timeString)
)

func Test_TimeFrom(t *testing.T) {
	ti := TimeFrom(timeValue)
	require.Equal(t, ti.Time, timeValue)
	require.True(t, ti.Valid, "TimeFrom() time.Time is invalid, but should be valid")
}

func Test_TimeString(t *testing.T) {
	ti := TimeFrom(timeValue)
	require.Equal(t, timeGoString, ti.String())

	ti.Valid = false
	require.Equal(t, "null", ti.String())
}

func Test_Time_UnmarshalJSON(t *testing.T) {
	happyPath := []struct {
		Name     string
		rawValue []byte
		errorMsg string
	}{
		{"success", timeJSON, ""},
		{"success null time", nullJSON, ""},
		{"success null uuid onempty string", []byte(`""`), ""},
		{"error on bad object", badObject, "json: cannot unmarshal  into Go value of type Time"},
		{"error on wrong type", intJSON, "json: cannot unmarshal float64 into Go value of type Time"},
	}

	for _, s := range happyPath {
		t.Run(s.Name, func(t *testing.T) {
			var ti Time
			err := json.Unmarshal(s.rawValue, &ti)
			requireEqualOrNilError(t, err, s.errorMsg)
		})
	}
}

func Test_Time_MarshalText(t *testing.T) {
	happyPath := []struct {
		Name      string
		timeValue Time
		txtValue  string
	}{
		{"success", TimeFrom(timeValue), timeString},
		{"success null time", Time{}, "null"},
	}

	for _, s := range happyPath {
		t.Run(s.Name, func(t *testing.T) {
			txt, err := s.timeValue.MarshalText()
			require.NoError(t, err)
			require.Equal(t, []byte(s.txtValue), txt)
		})
	}
}

func Test_Time_UnmarshalText(t *testing.T) {
	happyPath := []struct {
		Name     string
		rawValue []byte
		errorMsg string
	}{
		{"success", []byte(timeString), ""},
		{"success null time", []byte("null"), ""},
		{"error on bad text", []byte("hello world"), "parsing time \"hello world\" as \"2006-01-02T15:04:05Z07:00\": cannot parse \"hello world\" as \"2006\""},
	}

	for _, s := range happyPath {
		t.Run(s.Name, func(t *testing.T) {
			var ti Time
			err := ti.UnmarshalText(s.rawValue)
			requireEqualOrNilError(t, err, s.errorMsg)
		})
	}
}

func Test_Time_MarshalJSON(t *testing.T) {
	happyPath := []struct {
		Name      string
		timeValue Time
		jsonValue []byte
		errorMsg  string
	}{
		{"success non-empty time", TimeFrom(timeValue), timeJSON, ""},
		{"success null time", Time{Time: timeValue, Valid: false}, nullJSON, ""},
	}

	for _, s := range happyPath {
		t.Run(s.Name, func(t *testing.T) {
			data, err := json.Marshal(s.timeValue)
			requireEqualOrNilError(t, err, s.errorMsg)
			require.Equal(t, s.jsonValue, data)
		})
	}
}
