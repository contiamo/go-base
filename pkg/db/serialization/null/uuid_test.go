package null

import (
	"encoding/json"
	"testing"

	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/require"
)

var (
	uuidString = "a048b790-401a-4876-9488-1be1f7f2a0f4"
	uuidJSON   = []byte(`"` + uuidString + `"`)
	uuidBytes  = []byte(uuidString)
	uuidValue  = uuid.Must(uuid.FromString(uuidString))
)

func Test_UUIDFrom(t *testing.T) {
	var (
		u   UUID
		err error
	)

	u = UUIDFrom(uuidValue)
	require.Equal(t, u.UUID, uuidValue)
	require.True(t, u.Valid, "UUIDFrom() uuid.UUID is invalid, but should be valid")

	u, err = UUIDFromString(uuidString)
	require.NoError(t, err)
	require.Equal(t, u.UUID, uuidValue)
	require.True(t, u.Valid, "UUIDFromString() uuid.UUID is invalid, but should be valid")

	u, err = UUIDFromBytes(uuidValue.Bytes())
	require.NoError(t, err)
	require.Equal(t, u.UUID, uuidValue)
	require.True(t, u.Valid, "UUIDFromString() uuid.UUID is invalid, but should be valid")
}

func Test_UUID_MarshalJSON(t *testing.T) {
	happyPath := []struct {
		Name      string
		Value     UUID
		jsonValue []byte
		errorMsg  string
	}{
		{"success non-empty uuid", UUIDFrom(uuidValue), uuidJSON, ""},
		{"success null uuid", UUID{}, nullJSON, ""},
	}

	for _, s := range happyPath {
		t.Run(s.Name, func(t *testing.T) {
			data, err := json.Marshal(s.Value)
			requireEqualOrNilError(t, err, s.errorMsg)
			require.Equal(t, s.jsonValue, data)
		})
	}
}

func Test_UUID_UnmarshalJSON(t *testing.T) {
	happyPath := []struct {
		Name     string
		rawValue []byte
		errorMsg string
	}{
		{"success", uuidJSON, ""},
		{"success null uuid", nullJSON, ""},
		{"success null uuid onempty string", []byte(`""`), ""},
		{"error on bad object", badObject, "json: cannot unmarshal  into Go value of type UUID"},
		{"error on wrong type", intJSON, "json: cannot unmarshal float64 into Go value of type UUID"},
	}

	for _, s := range happyPath {
		t.Run(s.Name, func(t *testing.T) {
			var u UUID
			err := json.Unmarshal(s.rawValue, &u)
			requireEqualOrNilError(t, err, s.errorMsg)
		})
	}
}

func Test_UUID_MarshalText(t *testing.T) {
	happyPath := []struct {
		Name     string
		Value    UUID
		txtValue string
	}{
		{"success", UUIDFrom(uuidValue), uuidString},
		{"success null uuid", UUID{}, "null"},
	}

	for _, s := range happyPath {
		t.Run(s.Name, func(t *testing.T) {
			txt, err := s.Value.MarshalText()
			require.NoError(t, err)
			require.Equal(t, []byte(s.txtValue), txt)
		})
	}
}

func Test_UUID_UnmarshalText(t *testing.T) {
	happyPath := []struct {
		Name     string
		rawValue []byte
		errorMsg string
	}{
		{"success", uuidBytes, ""},
		{"success null uuid", []byte("null"), ""},
		{"error on bad text", []byte("hello world"), "uuid: incorrect UUID length: hello world"},
	}

	for _, s := range happyPath {
		t.Run(s.Name, func(t *testing.T) {
			var u UUID
			err := u.UnmarshalText(s.rawValue)
			requireEqualOrNilError(t, err, s.errorMsg)
		})
	}
}

// func TestTimeScanValue(t *testing.T) {
// 	var ti Time
// 	err := ti.Scan(timeValue)
// 	require.NoError(t, err)
// 	assertTime(t, ti, "scanned time")
// 	if v, err := ti.Value(); v != timeValue || err != nil {
// 		t.Error("bad value or err:", v, err)
// 	}

// 	var null Time
// 	err = null.Scan(nil)
// 	require.NoError(t, err)
// 	assertNullTime(t, null, "scanned null")
// 	if v, err := null.Value(); v != nil || err != nil {
// 		t.Error("bad value or err:", v, err)
// 	}

// 	var wrong Time
// 	err = wrong.Scan(int64(42))
// 	if err == nil {
// 		t.Error("expected error")
// 	}
// 	assertNullTime(t, wrong, "scanned wrong")
// }
