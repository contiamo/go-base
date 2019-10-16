package authorization

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_Valid(t *testing.T) {
	cases := []struct {
		name   string
		claims Claims
		valid  bool
	}{
		{
			name:   "Empty claims should not be valid",
			claims: Claims{},
			valid:  false,
		},
		{
			name: "If the user ID is set it's valid",
			claims: Claims{
				UserID: "this is a test id",
			},
			valid: true,
		},
		{
			name: "If the resource token ID is set it's valid",
			claims: Claims{
				ResourceTokenIDs: []string{"abc123"},
			},
			valid: true,
		},
		{
			name: "If the user ID and resource token ID are set it's valid",
			claims: Claims{
				UserID:           "this is a test id",
				ResourceTokenIDs: []string{"abc123"},
			},
			valid: true,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.valid, tc.claims.Valid())
		})
	}
}

func Test_SetGetClaims(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	claims := Claims{
		UserID:           "my user ID",
		ResourceTokenIDs: []string{"some", "other", "resource", "token"},
	}
	r = SetClaims(r, claims)
	extractedClaims, ok := GetClaims(r)
	require.True(t, ok)
	require.Equal(t, claims, extractedClaims)
}

func Test_Entities(t *testing.T) {
	claims := Claims{
		UserID:           "user-id",
		GroupIDs:         []string{"group-first", "group-second", "group-third"},
		ResourceTokenIDs: []string{"res-first", "res-second"},
	}

	require.Equal(
		t,
		[]string{
			"user-id",
			"group-first",
			"group-second",
			"group-third",
			"res-first",
			"res-second",
		},
		claims.Entities(),
	)
}

func Test_FromToClaims(t *testing.T) {
	claims := Claims{
		UserID:           "user-id",
		GroupIDs:         []string{"group-first", "group-second", "group-third"},
		ResourceTokenIDs: []string{"res-first", "res-second"},
	}
	jwtClaims, err := claims.ToClaims()
	require.NoError(t, err)
	require.Equal(t, claims.UserID, jwtClaims["sub"])
	require.Equal(t,
		[]interface{}{"group-first", "group-second", "group-third"},
		jwtClaims["groupIDs"],
	)
	require.Equal(t,
		[]interface{}{"res-first", "res-second"},
		jwtClaims["resourceTokenIDs"],
	)

	// edit the exported map and try to import its values
	jwtClaims["sub"] = "new-user-id"
	jwtClaims["groupIDs"] = []interface{}{"new-group-first", "new-group-second"}
	jwtClaims["resourceTokenIDs"] = []interface{}{"new-res-first", "new-res-second"}

	err = claims.FromClaimsMap(jwtClaims)
	require.NoError(t, err)
	require.Equal(t, jwtClaims["sub"], claims.UserID)
	require.Equal(
		t,
		[]string{"new-group-first", "new-group-second"},
		claims.GroupIDs,
	)
	require.Equal(
		t,
		[]string{"new-res-first", "new-res-second"},
		claims.ResourceTokenIDs,
	)
}
