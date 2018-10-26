package null

import (
	"testing"

	"github.com/stretchr/testify/require"
)

var (
	nullJSON    = []byte(`null`)
	invalidJSON = []byte(`:)`)
	intJSON     = []byte(`12345`)
	badObject   = []byte(`{"hello": "world"}`)
)

func requireEqualOrNilError(t *testing.T, err error, errorMsg string) {
	if errorMsg == "" {
		require.NoError(t, err)
	} else {
		require.EqualError(t, err, errorMsg)
	}
}
