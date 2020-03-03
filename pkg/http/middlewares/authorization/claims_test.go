package authorization

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/require"
)

var (
	userID = uuid.NewV4()
	testID = uuid.NewV4()
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
				UserID: userID,
			},
			valid: true,
		},
		{
			name: "If the resource token ID is set it's valid",
			claims: Claims{
				ResourceTokenIDs: []uuid.UUID{testID},
			},
			valid: true,
		},
		{
			name: "If the user ID and resource token ID are set it's valid",
			claims: Claims{
				UserID:           userID,
				ResourceTokenIDs: []uuid.UUID{testID},
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
		UserID:           userID,
		ResourceTokenIDs: []uuid.UUID{uuid.NewV4(), uuid.NewV4(), uuid.NewV4(), uuid.NewV4()},
	}
	r = SetClaims(r, claims)
	extractedClaims, ok := GetClaims(r)
	require.True(t, ok)
	require.Equal(t, claims, extractedClaims)
}

func Test_Entities(t *testing.T) {
	claims := Claims{
		UserID:           userID,
		GroupIDs:         []uuid.UUID{uuid.NewV4(), uuid.NewV4(), uuid.NewV4()},
		ResourceTokenIDs: []uuid.UUID{uuid.NewV4(), uuid.NewV4()},
	}

	require.Equal(
		t,
		[]uuid.UUID{
			userID,
			claims.GroupIDs[0],
			claims.GroupIDs[1],
			claims.GroupIDs[2],
			claims.ResourceTokenIDs[0],
			claims.ResourceTokenIDs[1],
		},
		claims.Entities(),
	)
}

func Test_FromToClaims(t *testing.T) {
	claims := Claims{
		UserID:           userID,
		GroupIDs:         []uuid.UUID{uuid.NewV4(), uuid.NewV4(), uuid.NewV4()},
		ResourceTokenIDs: []uuid.UUID{uuid.NewV4(), uuid.NewV4()},
	}
	jwtClaims, err := claims.ToClaims()
	require.NoError(t, err)
	require.Equal(t, claims.UserID.String(), jwtClaims["sub"])
	require.Equal(t,
		[]interface{}{
			claims.GroupIDs[0].String(),
			claims.GroupIDs[1].String(),
			claims.GroupIDs[2].String(),
		},
		jwtClaims["groupIDs"],
	)
	require.Equal(t,
		[]interface{}{
			claims.ResourceTokenIDs[0].String(),
			claims.ResourceTokenIDs[1].String(),
		},
		jwtClaims["resourceTokenIDs"],
	)

	newUserID := uuid.NewV4()
	groupID1 := uuid.NewV4()
	groupID2 := uuid.NewV4()
	resourceTokenID1 := uuid.NewV4()
	resourceTokenID2 := uuid.NewV4()

	// edit the exported map and try to import its values
	jwtClaims["sub"] = newUserID
	jwtClaims["groupIDs"] = []interface{}{groupID1.String(), groupID2.String()}
	jwtClaims["resourceTokenIDs"] = []interface{}{resourceTokenID1.String(), resourceTokenID2.String()}

	err = claims.FromClaimsMap(jwtClaims)
	require.NoError(t, err)
	require.Equal(t, newUserID, claims.UserID)
	require.Equal(
		t,
		[]uuid.UUID{groupID1, groupID2},
		claims.GroupIDs,
	)
	require.Equal(
		t,
		[]uuid.UUID{resourceTokenID1, resourceTokenID2},
		claims.ResourceTokenIDs,
	)
}

func Test_ClaimsMarshal(t *testing.T) {
	claims := Claims{
		UserID:           userID,
		GroupIDs:         []uuid.UUID{uuid.NewV4(), uuid.NewV4(), uuid.NewV4()},
		ResourceTokenIDs: []uuid.UUID{uuid.NewV4(), uuid.NewV4()},
	}

	bytes, err := json.Marshal(claims)
	require.NoError(t, err)

	var newClaims Claims
	err = json.Unmarshal(bytes, &newClaims)
	require.NoError(t, err)

	require.Equal(t, claims, newClaims)
}
