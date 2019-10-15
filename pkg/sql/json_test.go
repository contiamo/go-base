package sql

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

var (
	nilJSONStringArray     JSONStringArray
	emptyJSONStringArray   = JSONStringArray{}
	emptyArrayJSON         = []byte(`[]`)
	stringArrayValue       = JSONStringArray{"test1", "test2"}
	stringArrayJSON        = []byte(`["test1","test2"]`)
	invalidStringArrayJSON = []byte(`['test1', 'test2']`)

	jsonStringArrayScanStringHappyPath = []struct {
		name string
		val  JSONStringArray
		str  string
	}{
		{"array bytes", JSONStringArray{}, `[]`},
		{"single element bytes", JSONStringArray{"t"}, `["t"]`},
		{"multiple element bytes", JSONStringArray{"f", "1"}, `["f", "1"]`},
		{"with escape characters", JSONStringArray{"a\\b", "c d", ","}, `["a\\b", "c d", ","]`},
	}
)

func Test_JSONStringArray_UnmarshalJSON(t *testing.T) {
	happyPath := []struct {
		Name     string
		RawValue []byte
		Value    JSONStringArray
		ErrorMsg string
	}{
		{"success empty", emptyArrayJSON, emptyJSONStringArray, ""},
		{"success non-empty", stringArrayJSON, stringArrayValue, ""},
		{"nil on nil", nullJSON, nilJSONStringArray, ""},
		{"error on invalid json", invalidStringArrayJSON, nil, "invalid character '\\'' looking for beginning of value"},
	}

	for _, s := range happyPath {
		t.Run(s.Name, func(t *testing.T) {
			var u JSONStringArray
			err := json.Unmarshal(s.RawValue, &u)
			requireEqualOrNilError(t, err, s.ErrorMsg)
			require.Equal(t, s.Value, u)
		})
	}
}

// The following tests are adapted from lib/pq/array_test.go
func Test_JSONStringArray_MarshalJSON(t *testing.T) {
	happyPath := []struct {
		name      string
		value     JSONStringArray
		jsonValue []byte
		errorMsg  string
	}{
		{"success non-empty array", stringArrayValue, stringArrayJSON, ""},
		{"success empty array", emptyJSONStringArray, emptyArrayJSON, ""},
		{"success nil array", nilJSONStringArray, nullJSON, ""},
	}

	for _, s := range happyPath {
		t.Run(s.name, func(t *testing.T) {
			data, err := json.Marshal(s.value)
			requireEqualOrNilError(t, err, s.errorMsg)
			require.Equal(t, s.jsonValue, data)
		})
	}
}

func Test_JSONStringArray_ScanUnsupported(t *testing.T) {
	var arr JSONStringArray
	err := arr.Scan(true)

	require.EqualError(t, err, "sql: cannot convert bool to JSONStringArray")
}

func Test_JSONStringArray_ScanEmpty(t *testing.T) {
	var arr JSONStringArray
	err := arr.Scan(`[]`)
	require.NoError(t, err)
	require.Equal(t, emptyJSONStringArray, arr)
}

func Test_JSONStringArray_ScanNil(t *testing.T) {
	arr := JSONStringArray{"x", "x", "x"}
	err := arr.Scan(nil)

	require.NoError(t, err)
	require.Nil(t, arr)
}

func Test_JSONStringArray_ScanBytes(t *testing.T) {
	for _, s := range jsonStringArrayScanStringHappyPath {
		t.Run(s.name, func(t *testing.T) {
			bytes := []byte(s.str)
			arr := JSONStringArray{"x", "x", "x"}
			err := arr.Scan(bytes)
			require.NoError(t, err, "Expected no error for %q, got %v", bytes)
			require.Equal(t, arr, s.val)
		})
	}
}

func Test_JSONStringArray_ScanStrings(t *testing.T) {
	for _, s := range jsonStringArrayScanStringHappyPath {
		t.Run(s.name, func(t *testing.T) {
			arr := JSONStringArray{"x", "x", "x"}
			err := arr.Scan(s.str)

			require.NoError(t, err, "Expected no error for %q, got %v", s.str)
			require.Equal(t, arr, s.val)
		})
	}
}

func Test_JSONStringArray_Value(t *testing.T) {
	result, err := JSONStringArray(nil).Value()
	require.NoError(t, err)
	require.NotNil(t, result)

	result, err = JSONStringArray([]string{}).Value()
	require.NoError(t, err)
	require.Equal(t, emptyArrayJSON, result)

	result, err = JSONStringArray([]string{`a`, `\b`, `c"`, `d,e`}).Value()
	require.NoError(t, err)
	require.Equal(t, []byte(`["a","\\b","c\"","d,e"]`), result)
}

func Test_JSONMap_Scan(t *testing.T) {
	m := JSONMap{}
	err := m.Scan([]byte("null"))
	require.NoError(t, err)
	require.Nil(t, m)

	m = JSONMap(nil)
	err = m.Scan([]byte(`{"some":"value"}`))
	require.NoError(t, err)
	require.Equal(t, JSONMap{"some": "value"}, m)

	m = JSONMap(nil)
	err = m.Scan([]byte("not correct JSON"))
	require.Error(t, err)

	m = JSONMap(nil)
	err = m.Scan([]byte(`["item1", "item2"]`))
	require.Error(t, err)

	m = JSONMap(nil)
	err = m.Scan("not a massive of bytes")
	require.Error(t, err)
}

func Test_JSONMap_Value(t *testing.T) {
	result, err := JSONMap(nil).Value()
	require.NoError(t, err)
	require.Equal(t, []byte("null"), result)

	result, err = JSONMap(JSONMap{"some": "value"}).Value()
	require.NoError(t, err)
	require.Equal(t, []byte(`{"some":"value"}`), result)
}
