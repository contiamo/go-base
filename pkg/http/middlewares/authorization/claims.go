package authorization

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/contiamo/jwt"
	"github.com/google/uuid"
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

// Claims represents the expected claims that should be in a JWT sent to labs
//
// The IDP defined the token as
// type RequestToken struct {
// 	ID               string   `protobuf:"bytes,1,opt,name=ID,json=id,proto3" json:"ID,omitempty"`
// 	IssuedAt         float64  `protobuf:"fixed64,2,opt,name=IssuedAt,json=iat,proto3" json:"IssuedAt,omitempty"`
// 	NotBefore        float64  `protobuf:"fixed64,3,opt,name=NotBefore,json=nbf,proto3" json:"NotBefore,omitempty"`
// 	Expires          float64  `protobuf:"fixed64,4,opt,name=Expires,json=exp,proto3" json:"Expires,omitempty"`
// 	Issuer           string   `protobuf:"bytes,5,opt,name=Issuer,json=iss,proto3" json:"Issuer,omitempty"`
// 	UserID           string   `protobuf:"bytes,6,opt,name=UserID,json=sub,proto3" json:"UserID,omitempty"`
// 	UserName         string   `protobuf:"bytes,7,opt,name=UserName,json=name,proto3" json:"UserName,omitempty"`
// 	TenantID         string   `protobuf:"bytes,8,opt,name=TenantID,json=tenantID,proto3" json:"TenantID,omitempty"`
// 	Email            string   `protobuf:"bytes,9,opt,name=Email,json=email,proto3" json:"Email,omitempty"`
// 	RealmIDs         []string `protobuf:"bytes,10,rep,name=RealmIDs,json=realmIDs,proto3" json:"RealmIDs,omitempty"`
// 	GroupIDs         []string `protobuf:"bytes,11,rep,name=GroupIDs,json=groupIDs,proto3" json:"GroupIDs,omitempty"`
// 	ResourceTokenIDs []string `protobuf:"bytes,12,rep,name=ResourceTokenIDs,json=resourceTokenIDs,proto3" json:"ResourceTokenIDs,omitempty"`
// 	AllowedIPs       []string `protobuf:"bytes,13,rep,name=AllowedIPs,json=allowedIPs,proto3" json:"AllowedIPs,omitempty"`
// 	IsTenantAdmin    bool     `protobuf:"varint,14,opt,name=IsTenantAdmin,json=isTenantAdmin,proto3" json:"IsTenantAdmin,omitempty"`
// 	AdminRealmIDs    []string `protobuf:"bytes,15,rep,name=AdminRealmIDs,json=adminRealmIDs,proto3" json:"AdminRealmIDs,omitempty"`
// }
type Claims struct {
	ID               string    `json:"id"`
	IssuedAt         Timestamp `json:"iat"`
	NotBefore        Timestamp `json:"nbf"`
	Expires          Timestamp `json:"exp"`
	Issuer           string    `json:"iss"`
	UserID           string    `json:"sub"`
	UserName         string    `json:"name"`
	TenantID         string    `json:"tenantID"`
	Email            string    `json:"email"`
	RealmIDs         []string  `json:"realmIDs"`
	GroupIDs         []string  `json:"groupIDs"`
	ResourceTokenIDs []string  `json:"resourceTokenIDs"`
	AllowedIPs       []string  `json:"allowedIPs"`
	IsTenantAdmin    bool      `json:"isTenantAdmin"`
	AdminRealmIDs    []string  `json:"adminRealmIDs"`
	SourceToken      string    `json:-`
}

// Valid tests if the Claims object contains the minimal required information
// to be used for authorization checks.
func (a *Claims) Valid() bool {
	return a.UserID != "" || len(a.ResourceTokenIDs) > 0
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
	entities = append(entities, a.ResourceTokenIDs...)
	return entities
}

// GetClaims retrieves the Claims object from the request context
func GetClaims(r *http.Request) (Claims, bool) {
	claims, ok := r.Context().Value(authClaimsKey).(Claims)
	return claims, ok
}

// SetClaims add the Claims instance to the request Context
func SetClaims(r *http.Request, claims Claims) *http.Request {
	ctx := r.Context()
	ctx = context.WithValue(ctx, authClaimsKey, claims)
	return r.WithContext(ctx)
}
