package sql

import (
	"testing"

	"github.com/stretchr/testify/require"
)

var (
	nullJSON = []byte(`null`)
)

func requireEqualOrNilError(t *testing.T, err error, errorMsg string) {
	if errorMsg == "" {
		require.NoError(t, err)
	} else {
		require.EqualError(t, err, errorMsg)
	}
}
