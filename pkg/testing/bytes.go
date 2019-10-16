package testing

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

// ToJSONBytes converts any structure to a byte slice of its JSON representation
func ToJSONBytes(t *testing.T, obj interface{}) []byte {
	buf, err := json.Marshal(obj)
	require.NoError(t, err)
	// since all the endpoints serve `\n` it would be easier to have it here
	return append(buf, '\n')
}
