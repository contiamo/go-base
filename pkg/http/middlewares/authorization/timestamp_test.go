package authorization

import (
	"encoding/json"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

var (
	timeUnix                    = int64(1356124881)
	timeUnixString              = strconv.FormatInt(timeUnix, 10)
	timeUnixFloat               = float64(1356124881.0)
	timeUnixFloatWithNano       = float64(1356124881.10)
	timeUnixFloatWithNanoString = strconv.FormatFloat(timeUnixFloatWithNano, 'E', -1, 64)
	timeUnixFloatString         = strconv.FormatFloat(timeUnixFloat, 'E', -1, 64)
	timeString                  = "2012-12-21T21:21:21Z"
	timeJSON                    = []byte(`"` + timeString + `"`)
	timeValue, _                = time.Parse(time.RFC3339, timeString)
)

func Test_Timestamp_Unmarshal(t *testing.T) {

	scenarios := []struct {
		name  string
		input []byte
		value time.Time
	}{
		{"can parse RFC3339", timeJSON, timeValue},
		{"can parse numeric", []byte(timeUnixString), timeValue},
		{"can parse float value", []byte(timeUnixFloatString), timeValue},
		{"can parse numeric string", []byte(`"` + timeUnixString + `"`), timeValue},
	}

	for _, s := range scenarios {
		t.Run(s.name, func(t *testing.T) {
			var timestampValue Timestamp
			err := json.Unmarshal(s.input, &timestampValue)
			require.NoError(t, err)
			require.Equal(t, s.value, timestampValue.Time())
		})
	}
}

func Test_Timestamp_Unmarshal_Errors(t *testing.T) {

	scenarios := []struct {
		name  string
		input []byte
	}{
		{"occurs for null value", []byte(`null`)},
		{"occurs for non-time values", []byte(`"this-tis-not-a-time"`)},
		{"occurs if string must be numeric or rfc 3339", []byte(`2012-12-21 21:21:21Z"`)},
	}

	for _, s := range scenarios {
		t.Run(s.name, func(t *testing.T) {
			var timestampValue Timestamp

			require.Error(t, json.Unmarshal(s.input, &timestampValue))
		})
	}
}

func Test_Timestamp_Unmarshal_FloatWithNanos(t *testing.T) {
	var timestampValue Timestamp

	require.NoError(t, json.Unmarshal([]byte(timeUnixFloatWithNanoString), &timestampValue))
	require.InDelta(t, timeValue.UnixNano(), timestampValue.Time().UnixNano(), 1e+09)
}

func Test_Timestamp_Marshal_Returns_Unix_Timestamp(t *testing.T) {
	value := FromTime(timeValue)
	jsonStr, err := json.Marshal(&value)
	require.NoError(t, err)
	require.Equal(t, []byte(timeUnixString), jsonStr)
}
