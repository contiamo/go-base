package authorization

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	jwt "github.com/golang-jwt/jwt/v4"
	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/require"
)

var (
	now                   = time.Now()
	privateKey, publicKey = generatekeys()
	jwtClaims             = map[string]interface{}{
		"email":         "root@localhost",
		"groupIDs":      []string{},
		"id":            "ef81f4b0-0b96-4874-9315-e5e9d01eb481",
		"iat":           now.Add(-5 * time.Second).Unix(),
		"exp":           now.Add(time.Hour).Unix(),
		"nbf":           now.Add(-5 * time.Second).Unix(),
		"iss":           "idp",
		"name":          "root",
		"realmIDs":      []string{},
		"sub":           "2fc15360-6ee9-4bbb-be4a-e3ea319324d3",
		"tenantID":      "7cfa3fa7-92cf-4410-8c99-04ec8e018c15",
		"isTenantAdmin": false,
		"adminRealmIDs": []string{},
		"extraField":    "this should be ignored",
	}
	token = getToken(jwtClaims)
)

func Test_middleware(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Freeze time
	TimeFunc = func() time.Time {
		return now
	}

	var claims Claims
	require.NoError(t, claims.FromClaimsMap(jwtClaims))
	claims.SourceToken = token

	client := http.Client{}
	authMiddleware := NewMiddleware("X-Auth-Token", publicKey)

	t.Run("JWT parses and the claims contain the original source token", func(t *testing.T) {
		ts := httptest.NewServer(authMiddleware.WrapHandler(getJWTAuthzHandlerFunc(t, claims)))
		defer ts.Close()

		req, err := http.NewRequest(http.MethodGet, ts.URL, nil)
		req = req.WithContext(ctx)
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
		req = req.WithContext(ctx)
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
		req = req.WithContext(ctx)
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
		req = req.WithContext(ctx)
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
		req = req.WithContext(ctx)
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
		// claims must have sub, exp, iat, and nbf
		claimsMap := map[string]interface{}{
			"name":  "John Doe",
			"admin": true,
		}
		require.NoError(t, claims.FromClaimsMap(claimsMap))
		claims.SourceToken = getToken(claimsMap)

		ts := httptest.NewServer(authMiddleware.WrapHandler(getEmptyHandlerFunc()))
		defer ts.Close()

		req, err := http.NewRequest(http.MethodGet, ts.URL, nil)
		req = req.WithContext(ctx)
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

	t.Run("Unauthorized if token is is too old: expired", func(t *testing.T) {
		var claims Claims
		claimsMap := map[string]interface{}{
			"sub": uuid.NewV4(),
			"iat": now,
			"exp": now.Add(-5 * time.Second).Unix(),
			"nbf": now.Add(-10 * time.Second).Unix(),
		}
		require.NoError(t, claims.FromClaimsMap(claimsMap))

		ts := httptest.NewServer(authMiddleware.WrapHandler(getEmptyHandlerFunc()))
		defer ts.Close()

		req, err := http.NewRequest(http.MethodGet, ts.URL, nil)
		req = req.WithContext(ctx)
		require.NoError(t, err)
		req.Header.Add("X-Auth-Token", getToken(claimsMap))

		res, err := client.Do(req)
		require.NoError(t, err)
		defer res.Body.Close()

		require.Equal(t, http.StatusUnauthorized, res.StatusCode)

		b, err := ioutil.ReadAll(res.Body)
		require.NoError(t, err)

		require.Equal(t, "Unauthorized\n", string(b))
	})

	t.Run("Unauthorized if token azp and iss mismatch", func(t *testing.T) {
		var claims Claims
		// claims are missing "sub" and "resourceTokenIDs"
		claimsMap := map[string]interface{}{
			"sub": uuid.NewV4(),
			"iat": now,
			"exp": now.Add(10 * time.Second).Unix(),
			"nbf": now.Add(-2 * time.Second).Unix(),
			"iss": "service-a",
			"azp": "service-b",
		}
		require.NoError(t, claims.FromClaimsMap(claimsMap))

		ts := httptest.NewServer(authMiddleware.WrapHandler(getEmptyHandlerFunc()))
		defer ts.Close()

		req, err := http.NewRequest(http.MethodGet, ts.URL, nil)
		req = req.WithContext(ctx)
		require.NoError(t, err)
		req.Header.Add("X-Auth-Token", getToken(claimsMap))

		res, err := client.Do(req)
		require.NoError(t, err)
		defer res.Body.Close()

		require.Equal(t, http.StatusUnauthorized, res.StatusCode)

		b, err := ioutil.ReadAll(res.Body)
		require.NoError(t, err)

		require.Equal(t, "Unauthorized\n", string(b))
	})
}

func Test_sanitizeHeaderValue(t *testing.T) {
	cases := [][2]string{
		{"   ", ""},
		{"a", "****"},
		{"abc", "a****"},
		{"abcd", "ab****"},
		{"abcd..", "abc****"},
		{"a.bc.d", "a.b****"},
		{
			"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ.SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c",
			"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.****",
		},
	}
	for _, tc := range cases {
		t.Run(
			fmt.Sprintf("scrubbing %q should produce %s", tc[0], tc[1]),
			func(t *testing.T) {
				out := sanitizeHeaderValue(tc[0])
				require.Equal(t, tc[1], out)
			})
	}
}

func generatekeys() (privateKey *rsa.PrivateKey, publicKey *rsa.PublicKey) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
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
