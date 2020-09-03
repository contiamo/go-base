package tokens

import (
	"testing"

	"github.com/contiamo/go-base/pkg/config"
	jwt "github.com/dgrijalva/jwt-go"
	"github.com/stretchr/testify/require"
)

func TestTokenCreatorCreateProjectAdmin(t *testing.T) {
	t.Run("creates a valid signed JWT token", func(t *testing.T) {
		cfg := config.JWT{
			PrivateKeyPath: "./testdata/idp.key",
			PublicKeyPath:  "./testdata/idp.crt",
		}

		privateKey, err := cfg.GetPrivateKey()
		require.NoError(t, err)
		publicKey, err := cfg.GetPublicKey()
		require.NoError(t, err)

		tc := NewCreator(privateKey, 0)
		tokenStr, err := tc.CreateProjectAdmin("project", "background-task")
		require.NoError(t, err)
		require.NotEmpty(t, tokenStr)

		keyFunc := func(*jwt.Token) (interface{}, error) { return publicKey, nil }
		var claims token
		token, err := jwt.ParseWithClaims(tokenStr, &claims, keyFunc)
		require.NoError(t, err)
		require.NotNil(t, token)
		require.Equal(t, "hub", claims.Issuer)
		require.Equal(t, "@hub", claims.UserName)
		require.Equal(t, []string{"project"}, claims.AdminRealmIDs)
		require.Equal(t, []string{"background-task"}, claims.AuthenticationMethodReferences)
	})

	t.Run("fails if the private key is empty", func(t *testing.T) {
		tc := NewCreator(nil, 0)
		tokenStr, err := tc.CreateProjectAdmin("project", "background-task")
		require.Empty(t, tokenStr)
		require.Error(t, err)
		require.Equal(t, ErrNoPrivateKeySpecified.Error(), err.Error())
	})
}
