package authorization

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func Test_Valid(t *testing.T) {
	// Freeze time
	now := time.Now()
	TimeFunc = func() time.Time {
		return now
	}

	cases := []struct {
		name   string
		claims Claims
		err    error
	}{
		{
			name:   "Empty claims should not be valid",
			claims: Claims{},
			err:    ErrMissingSub,
		},
		{
			name: "Invalid if used before the iat",
			claims: Claims{
				UserID:    "this is a test id",
				IssuedAt:  FromTime(now.Add(2 * time.Second)),
				Expires:   FromTime(now.Add(5 * time.Second)),
				NotBefore: FromTime(now.Add(-1 * time.Second)),
			},
			err: ErrTooSoon,
		},
		{
			name: "Invalid if used before the nbf",
			claims: Claims{
				UserID:    "this is a test id",
				IssuedAt:  FromTime(now.Add(-1 * time.Second)),
				Expires:   FromTime(now.Add(5 * time.Second)),
				NotBefore: FromTime(now.Add(1 * time.Second)),
			},
			err: ErrTooEarly,
		},
		{
			name: "Invalid if used after exp",
			claims: Claims{
				UserID:    "this is a test id",
				IssuedAt:  FromTime(now.Add(-5 * time.Second)),
				Expires:   FromTime(now.Add(-1 * time.Second)),
				NotBefore: FromTime(now.Add(-5 * time.Second)),
			},
			err: ErrExpiration,
		},
		{
			name: "minimally valid claims requires sub, iat, exp, and nbf",
			claims: Claims{
				UserID:    "this is a test id",
				IssuedAt:  FromTime(now.Add(-1 * time.Second)),
				Expires:   FromTime(now.Add(time.Second)),
				NotBefore: FromTime(now.Add(-1 * time.Second)),
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.claims.Validate(), tc.err)
			require.Equal(t, tc.err == nil, tc.claims.Valid())
		})
	}
}

func Test_SetGetClaims(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	claims := Claims{
		UserID: "my user ID",
	}
	r = SetClaims(r, claims)
	extractedClaims, ok := GetClaims(r)
	require.True(t, ok)
	require.Equal(t, claims, extractedClaims)
}

func Test_Entities(t *testing.T) {
	claims := Claims{
		UserID:   "user-id",
		GroupIDs: []string{"group-first", "group-second", "group-third"},
	}

	require.Equal(
		t,
		[]string{
			"user-id",
			"group-first",
			"group-second",
			"group-third",
		},
		claims.Entities(),
	)
}

func Test_FromToClaims(t *testing.T) {
	claims := Claims{
		UserID:   "user-id",
		GroupIDs: []string{"group-first", "group-second", "group-third"},
	}
	jwtClaims, err := claims.ToClaims()
	require.NoError(t, err)
	require.Equal(t, claims.UserID, jwtClaims["sub"])
	require.Equal(t,
		[]interface{}{"group-first", "group-second", "group-third"},
		jwtClaims["groupIDs"],
	)

	// edit the exported map and try to import its values
	jwtClaims["sub"] = "new-user-id"
	jwtClaims["groupIDs"] = []interface{}{"new-group-first", "new-group-second"}

	err = claims.FromClaimsMap(jwtClaims)
	require.NoError(t, err)
	require.Equal(t, jwtClaims["sub"], claims.UserID)
	require.Equal(
		t,
		[]string{"new-group-first", "new-group-second"},
		claims.GroupIDs,
	)
}
