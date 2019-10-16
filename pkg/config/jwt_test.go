package config

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestJWTGetPublicKey(t *testing.T) {
	t.Run("Returns error when the path is empty", func(t *testing.T) {
		cfg := JWT{}
		key, err := cfg.GetPublicKey()
		require.Nil(t, key)
		require.Error(t, err)
		require.Equal(t, "path to the public key file is empty", err.Error())
	})
	t.Run("Returns error when the path is not PEM certificate", func(t *testing.T) {
		cfg := JWT{PublicKeyPath: "./testdata/password"}
		key, err := cfg.GetPublicKey()
		require.Nil(t, key)
		require.Error(t, err)
		require.Equal(t, "Invalid Key: Key must be PEM encoded PKCS1 or PKCS8 private key", err.Error())
	})
	t.Run("Returns error when the path is not found", func(t *testing.T) {
		cfg := JWT{PublicKeyPath: "./testdata/invalid"}
		key, err := cfg.GetPublicKey()
		require.Nil(t, key)
		require.Error(t, err)
		require.Equal(t, "can not read the public key file `./testdata/invalid`: open ./testdata/invalid: no such file or directory", err.Error())
	})

	t.Run("Returns key when the path is valid", func(t *testing.T) {
		cfg := JWT{PublicKeyPath: "./testdata/test.crt"}
		key, err := cfg.GetPublicKey()
		require.NoError(t, err)
		require.NotNil(t, key)
	})
}
