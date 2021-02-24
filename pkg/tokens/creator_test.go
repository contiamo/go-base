package tokens

import (
	"testing"

	"github.com/contiamo/go-base/v2/pkg/config"
	jwt "github.com/dgrijalva/jwt-go"
	"github.com/stretchr/testify/require"
)

func TestTokenCreatorCreate(t *testing.T) {
	t.Run("creates a valid signed JWT token", func(t *testing.T) {
		cfg := config.JWT{
			PrivateKeyPath: "./testdata/idp.key",
			PublicKeyPath:  "./testdata/idp.crt",
		}

		privateKey, err := cfg.GetPrivateKey()
		require.NoError(t, err)
		publicKey, err := cfg.GetPublicKey()
		require.NoError(t, err)

		tc := NewCreator("go-base", privateKey, 0)
		tokenStr, err := tc.Create("background-task", Options{ProjectID: "project"})
		require.NoError(t, err)
		require.NotEmpty(t, tokenStr)

		keyFunc := func(*jwt.Token) (interface{}, error) { return publicKey, nil }
		var claims token
		token, err := jwt.ParseWithClaims(tokenStr, &claims, keyFunc)
		require.NoError(t, err)
		require.NotNil(t, token)
		require.Equal(t, "go-base", claims.Issuer)
		require.Equal(t, "@go-base", claims.UserName)
		require.Equal(t, []string{"project"}, claims.RealmIDs)
		require.Equal(t, []string{"project"}, claims.AdminRealmIDs)
		require.Equal(t, []string{"background-task"}, claims.AuthenticationMethodReferences)
		require.Equal(t, "go-base", claims.AuthorizedParty)
		require.Equal(t, "", claims.Audience)
	})

	t.Run("creates a valid signed JWT token with audience", func(t *testing.T) {
		cfg := config.JWT{
			PrivateKeyPath: "./testdata/idp.key",
			PublicKeyPath:  "./testdata/idp.crt",
		}

		privateKey, err := cfg.GetPrivateKey()
		require.NoError(t, err)
		publicKey, err := cfg.GetPublicKey()
		require.NoError(t, err)

		tc := NewCreator("go-base", privateKey, 0)
		tokenStr, err := tc.Create("background-task", Options{Audience: "hub"})
		require.NoError(t, err)
		require.NotEmpty(t, tokenStr)

		keyFunc := func(*jwt.Token) (interface{}, error) { return publicKey, nil }
		var claims token
		token, err := jwt.ParseWithClaims(tokenStr, &claims, keyFunc)
		require.NoError(t, err)
		require.NotNil(t, token)
		require.Equal(t, "go-base", claims.Issuer)
		require.Equal(t, "@go-base", claims.UserName)
		require.Equal(t, []string{}, claims.RealmIDs)
		require.Equal(t, []string{}, claims.AdminRealmIDs)
		require.Equal(t, []string{"background-task"}, claims.AuthenticationMethodReferences)
		require.Equal(t, "go-base", claims.AuthorizedParty)
		require.Equal(t, "hub", claims.Audience)
	})

	t.Run("fails if the private key is empty", func(t *testing.T) {
		tc := NewCreator("go-base", nil, 0)
		tokenStr, err := tc.Create("background-task", Options{ProjectID: "project"})
		require.Empty(t, tokenStr)
		require.Error(t, err)
		require.Equal(t, ErrNoPrivateKeySpecified.Error(), err.Error())
	})
}
