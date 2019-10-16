package authorization

import (
	"crypto/rand"
	"crypto/rsa"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	jwt "github.com/dgrijalva/jwt-go"
	"github.com/stretchr/testify/require"
)

var (
	privateKey, publicKey = generatekeys()
	jwtClaims             = map[string]interface{}{
		"allowedIPs":       []string{},
		"email":            "root@localhost",
		"groupIDs":         []string{},
		"id":               "ef81f4b0-0b96-4874-9315-e5e9d01eb481",
		"iat":              time.Now().Unix(),
		"exp":              time.Now().Add(time.Hour).Unix(),
		"nbf":              time.Now().Unix(),
		"iss":              "idp",
		"name":             "root",
		"realmIDs":         []string{},
		"resourceTokenIDs": []string{},
		"sub":              "2fc15360-6ee9-4bbb-be4a-e3ea319324d3",
		"tenantID":         "7cfa3fa7-92cf-4410-8c99-04ec8e018c15",
		"isTenantAdmin":    false,
		"adminRealmIDs":    []string{},
		"extraField":       "this should be ignored",
	}
	token = getToken(jwtClaims)
)

func generatekeys() (privateKey *rsa.PrivateKey, publicKey *rsa.PublicKey) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		panic(err)
	}

	return privateKey, &privateKey.PublicKey
}

func getToken(claimsMap map[string]interface{}) string {
	token, err := jwt.
		NewWithClaims(jwt.SigningMethodRS512, jwt.MapClaims(claimsMap)).
		SignedString(privateKey)
	if err != nil {
		panic(err)
	}

	return token
}

func getJWTAuthzHandlerFunc(t *testing.T, claims Claims) http.HandlerFunc {
	fn := func(w http.ResponseWriter, r *http.Request) {
		parsed, ok := GetClaims(r)
		require.True(t, ok)
		require.Equal(t, claims, parsed)
		_, err := w.Write([]byte("OK!"))
		require.NoError(t, err)
	}

	return http.HandlerFunc(fn)
}

func getEmptyHandlerFunc() http.HandlerFunc {
	fn := func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("Ok!"))
	}

	return http.HandlerFunc(fn)
}

func Test_middleware(t *testing.T) {
	var claims Claims
	require.NoError(t, claims.FromClaimsMap(jwtClaims))
	claims.SourceToken = token

	client := http.Client{}
	authMiddleware := NewMiddleware("X-Auth-Token", publicKey)

	t.Run("JWT parses and the claims contain the original source token", func(t *testing.T) {
		ts := httptest.NewServer(authMiddleware.WrapHandler(getJWTAuthzHandlerFunc(t, claims)))
		defer ts.Close()

		req, err := http.NewRequest(http.MethodGet, ts.URL, nil)
		require.NoError(t, err)
		req.Header.Add("X-Auth-Token", token)

		res, err := client.Do(req)
		require.NoError(t, err)
		defer res.Body.Close()

		require.Equal(t, http.StatusOK, res.StatusCode)

		b, err := ioutil.ReadAll(res.Body)
		require.NoError(t, err)
		require.Equal(t, "OK!", string(b))
	})

	t.Run("Unauthorized if the signature is incorrect", func(t *testing.T) {
		_, publicKey := generatekeys() // the token was signed with a different key
		authMiddleware := NewMiddleware("X-Auth-Token", publicKey)

		ts := httptest.NewServer(authMiddleware.WrapHandler(getJWTAuthzHandlerFunc(t, claims)))
		defer ts.Close()

		req, err := http.NewRequest(http.MethodGet, ts.URL, nil)
		require.NoError(t, err)
		req.Header.Add("X-Auth-Token", token)

		res, err := client.Do(req)
		require.NoError(t, err)
		defer res.Body.Close()

		require.Equal(t, http.StatusUnauthorized, res.StatusCode)

		b, err := ioutil.ReadAll(res.Body)
		require.NoError(t, err)
		require.Equal(t, "Unauthorized\n", string(b))
	})

	t.Run("Unauthorized if missing token", func(t *testing.T) {
		ts := httptest.NewServer(authMiddleware.WrapHandler(getEmptyHandlerFunc()))
		defer ts.Close()

		req, err := http.NewRequest(http.MethodGet, ts.URL, nil)
		require.NoError(t, err)

		res, err := client.Do(req)
		require.NoError(t, err)
		defer res.Body.Close()

		require.Equal(t, http.StatusUnauthorized, res.StatusCode)

		b, err := ioutil.ReadAll(res.Body)
		require.NoError(t, err)
		require.Equal(t, "Unauthorized\n", string(b))
	})

	t.Run("Unauthorized if token is malformed: too short", func(t *testing.T) {
		ts := httptest.NewServer(authMiddleware.WrapHandler(getEmptyHandlerFunc()))
		defer ts.Close()

		req, err := http.NewRequest(http.MethodGet, ts.URL, nil)
		require.NoError(t, err)
		req.Header.Add("X-Auth-Token", "abcs.123")

		res, err := client.Do(req)
		require.NoError(t, err)
		defer res.Body.Close()

		require.Equal(t, http.StatusUnauthorized, res.StatusCode)

		b, err := ioutil.ReadAll(res.Body)
		require.NoError(t, err)
		require.Equal(t, "Unauthorized\n", string(b))
	})

	t.Run("Unauthorized if token is malformed: bad payload", func(t *testing.T) {
		ts := httptest.NewServer(authMiddleware.WrapHandler(getEmptyHandlerFunc()))
		defer ts.Close()

		req, err := http.NewRequest(http.MethodGet, ts.URL, nil)
		require.NoError(t, err)
		req.Header.Add("X-Auth-Token", "abcs.123.efg")

		res, err := client.Do(req)
		require.NoError(t, err)
		defer res.Body.Close()

		require.Equal(t, http.StatusUnauthorized, res.StatusCode)

		b, err := ioutil.ReadAll(res.Body)
		require.NoError(t, err)
		require.Equal(t, "Unauthorized\n", string(b))
	})

	t.Run("Unauthorized if token is malformed: missing info", func(t *testing.T) {
		var claims Claims
		// claims are missing "sub" and "resourceTokenIDs"
		claimsMap := map[string]interface{}{
			"name":  "John Doe",
			"admin": true,
		}
		require.NoError(t, claims.FromClaimsMap(claimsMap))
		claims.SourceToken = getToken(claimsMap)

		ts := httptest.NewServer(authMiddleware.WrapHandler(getEmptyHandlerFunc()))
		defer ts.Close()

		req, err := http.NewRequest(http.MethodGet, ts.URL, nil)
		require.NoError(t, err)
		req.Header.Add("X-Auth-Token", claims.SourceToken)

		res, err := client.Do(req)
		require.NoError(t, err)
		defer res.Body.Close()

		require.Equal(t, http.StatusUnauthorized, res.StatusCode)

		b, err := ioutil.ReadAll(res.Body)
		require.NoError(t, err)

		require.Equal(t, "Unauthorized\n", string(b))
	})
}
