package authorization

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/contiamo/jwt"
	uuid "github.com/satori/go.uuid"
	"github.com/sirupsen/logrus"
)

// authContextKey is an unexported type for keys defined in middleware.
// This prevents collisions with keys defined in other packages.
type authContextKey string

func (c authContextKey) String() string {
	return authContextKeyPrefix + string(c)
}

var (
	authContextKeyPrefix = "com.contiamo.labs.auth.contextKey."
	// proxyAuthTokenClaimsKey is the context key used to set/access the JWT claims
	authClaimsKey = authContextKey("DatastoreClaims")
	// DataStoreClaims used for setting the service itself as an author of a record
	DataStoreClaims = Claims{
		UserID:   uuid.Nil.String(),
		UserName: "datastore",
	}
)

// Claims represents the expected claims that should be in JWT claims of an X-Request-Token
type Claims struct {
	// standard oidc claims
	ID        string    `json:"id"`
	Issuer    string    `json:"iss"`
	IssuedAt  Timestamp `json:"iat"`
	NotBefore Timestamp `json:"nbf"`
	Expires   Timestamp `json:"exp"`
	Audience  string    `json:"aud,omitempty"`

	UserID   string `json:"sub"`
	UserName string `json:"name"`
	Email    string `json:"email"`

	// Contiamo specific claims
	TenantID      string   `json:"tenantID"`
	RealmIDs      []string `json:"realmIDs"`
	GroupIDs      []string `json:"groupIDs"`
	AllowedIPs    []string `json:"allowedIPs"`
	IsTenantAdmin bool     `json:"isTenantAdmin"`
	AdminRealmIDs []string `json:"adminRealmIDs"`

	AuthenticationMethodReferences []string `json:"amr"`
	// AuthorizedParty is used to indicate that the request is authorizing as a
	// service request, giving it super-admin privileges to completely any request.
	// This replaces the "project admin" behavior of the current tokens.
	AuthorizedParty string `json:"azp,omitempty"`

	// SourceToken is for internal usage only
	SourceToken string `json:"-"`
}

// Valid tests if the Claims object contains the minimal required information
// to be used for authorization checks.
//
// Deprecated: Use the Validate method to get a precise error message. This
// method remains for backward compatibility.
func (a *Claims) Valid() bool {

	return a.Validate() == nil
}

// Validate verifies the token claims.
func (a Claims) Validate() (err error) {
	defer func() {
		if err != nil {
			logrus.WithError(err).Error("claims validation error")
		}
	}()

	now := TimeFunc()

	// this validation is specific to contiamo
	if a.UserID == "" {
		return ErrMissingSub
	}

	// the middleware parsing will generally run this validation, but
	// adding it here marks the exp as a required claim
	if !a.VerifyExpiresAt(now, true) {
		return ErrExpiration
	}

	// the middleware parsing will generally run this validation, but
	// adding it here marks the nbf as a required claim
	if !a.VerifyNotBefore(now, true) {
		return ErrTooEarly
	}

	// the middleware parsing will generally run this validation, but
	// adding it here marks the iat as a required claim
	if !a.VerifyIssuedAt(now, true) {
		return ErrTooSoon
	}

	// this validation is specific to contiamo
	if !a.VerifyAuthorizedParty() {
		return ErrInvalidParty
	}

	return nil
}

// FromClaimsMap loads the claim information from a jwt.Claims object, this is a simple
// map[string]interface{}
func (a *Claims) FromClaimsMap(claims jwt.Claims) error {
	bs, err := json.Marshal(claims)
	if err != nil {
		return err
	}
	return json.Unmarshal(bs, a)
}

// ToClaims encodes the token as jwt.Claims
func (a *Claims) ToClaims() (jwt.Claims, error) {
	claimBytes, err := json.Marshal(a)
	if err != nil {
		return nil, err
	}
	claims := make(jwt.Claims)
	if err = json.Unmarshal(claimBytes, &claims); err != nil {
		return nil, err
	}
	return claims, nil
}

// ToJWT encodes the token to a valid jwt
func (a *Claims) ToJWT(privateKey interface{}) (string, error) {
	claims, err := a.ToClaims()
	if err != nil {
		return "", err
	}
	return jwt.CreateToken(claims, privateKey)
}

// Entities returns a slice of the entity ids that the auth claims contains.  These are ids
// that permissions may be assigned to. Currently, this is the UserID, GroupIDs, and ResourceTokenIDs
func (a *Claims) Entities() (entities []string) {
	entities = append(entities, a.UserID)
	entities = append(entities, a.GroupIDs...)
	return entities
}

// VerifyAudience compares the aud claim against cmp.
func (a Claims) VerifyAudience(cmp string, required bool) bool {
	if a.Audience == "" {
		return !required
	}

	return a.Audience == cmp
}

// VerifyExpiresAt compares the exp claim against the cmp time.
func (a Claims) VerifyExpiresAt(cmp time.Time, required bool) bool {
	if a.Expires.time.IsZero() {
		return !required
	}

	return cmp.Before(a.Expires.time)
}

// VerifyNotBefore compares the nbf claim against the cmp time.
func (a Claims) VerifyNotBefore(cmp time.Time, required bool) bool {
	if a.NotBefore.time.IsZero() {
		return !required
	}

	return cmp.After(a.NotBefore.time) || cmp.Equal(a.NotBefore.time)
}

// VerifyIssuedAt compares the iat claim against the cmp time.
func (a Claims) VerifyIssuedAt(cmp time.Time, required bool) bool {
	if a.IssuedAt.time.IsZero() {
		return !required
	}

	return cmp.After(a.IssuedAt.time) || cmp.Equal(a.IssuedAt.time)
}

// VerifyIssuer compares the iss claim against cmp.
func (a Claims) VerifyIssuer(cmp string, required bool) bool {
	if a.Issuer == "" {
		return !required
	}

	return a.Issuer == cmp
}

// VerifyAuthorizedParty verify that azp matches the iss value, if set.
func (a Claims) VerifyAuthorizedParty() bool {
	if a.AuthorizedParty == "" {
		return true
	}
	return a.VerifyIssuer(a.AuthorizedParty, true)
}

// GetClaims retrieves the Claims object from the request context
func GetClaims(r *http.Request) (Claims, bool) {
	claims, ok := GetClaimsFromCtx(r.Context())
	return claims, ok
}

// GetClaimsFromCtx retrieves the Claims object from the given context
func GetClaimsFromCtx(ctx context.Context) (Claims, bool) {
	claims, ok := ctx.Value(authClaimsKey).(Claims)
	return claims, ok
}

// SetClaims add the Claims instance to the request Context
func SetClaims(r *http.Request, claims Claims) *http.Request {
	ctx := r.Context()
	ctx = context.WithValue(ctx, authClaimsKey, claims)
	return r.WithContext(ctx)
}
