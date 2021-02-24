package tokens

import (
	"crypto/rsa"
	"errors"
	"time"

	jwt "github.com/dgrijalva/jwt-go"

	uuid "github.com/satori/go.uuid"
)

var (
	// ErrNoPrivateKeySpecified occurs when the private key was not set
	// and there was an attempt to create a token
	ErrNoPrivateKeySpecified = errors.New("private key is nil")
)

type token struct {
	ID        string  `json:"id"`
	Issuer    string  `json:"iss"`
	IssuedAt  float64 `json:"iat"`
	NotBefore float64 `json:"nbf"`
	Expires   float64 `json:"exp"`
	Audience  string  `json:"aud,omitempty"`

	UserID   string `json:"sub"`
	UserName string `json:"name"`
	Email    string `json:"email"`

	TenantID                       string   `json:"tenantID"`
	RealmIDs                       []string `json:"realmIDs"`
	GroupIDs                       []string `json:"groupIDs"`
	AllowedIPs                     []string `json:"allowedIPs,omitempty"`
	IsTenantAdmin                  bool     `json:"isTenantAdmin"`
	AdminRealmIDs                  []string `json:"adminRealmIDs"`
	AuthenticationMethodReferences []string `json:"amr"`
	// AuthorizedParty is used to indicate that the request is authorizing as a
	// service request, giving it super-admin privileges to completely any request.
	// This replaces the "project admin" behavior of the current tokens.
	AuthorizedParty string `json:"azp,omitempty"`
}

func (token) Valid() error {
	return nil
}

// Options control the value or the generation of the claims in the resulting token.
// All values are optional and the empty value will be ignored.
type Options struct {
	// Audience is a name of the service that receives the request. Other
	// services should not validate tokens intended for other services.
	Audience string
	// ProjectID is the UUID string for a project that the token should be
	// considered a member and an admin of. This value is deprecated, but
	// exists for backwards compatibility during the transition to `azp`.
	ProjectID string
}

// Creator creates all kinds of signed tokens for the background tasks
type Creator interface {
	// Create creates a signed token that can be used for interservice communication.
	Create(reference string, opts Options) (string, error)
}

// NewCreator creates a new token creator for tasks
func NewCreator(issuer string, privateKey *rsa.PrivateKey, lifetime time.Duration) Creator {
	if lifetime < 5 {
		lifetime = 5 * time.Second
	}
	return &tokenCreator{
		issuer:   issuer,
		jwtKey:   privateKey,
		lifetime: lifetime,
	}
}

type tokenCreator struct {
	issuer   string
	jwtKey   *rsa.PrivateKey
	lifetime time.Duration
}

func (t *tokenCreator) Create(reference string, opts Options) (string, error) {
	if t.jwtKey == nil {
		return "", ErrNoPrivateKeySpecified
	}

	now := time.Now()
	maxSkew := 5 * time.Second
	requestToken := token{
		ID:                             uuid.NewV4().String(),
		Issuer:                         t.issuer,
		IssuedAt:                       float64(now.Unix()),
		NotBefore:                      float64(now.Add(-1 * maxSkew).Unix()),
		Expires:                        float64(now.Add(t.lifetime).Unix()),
		UserID:                         uuid.Nil.String(),
		UserName:                       "@" + t.issuer,
		Email:                          t.issuer + "@contiamo.com",
		RealmIDs:                       []string{},
		GroupIDs:                       []string{},
		AllowedIPs:                     []string{},
		AdminRealmIDs:                  []string{},
		AuthenticationMethodReferences: []string{reference},
		AuthorizedParty:                t.issuer,
		Audience:                       opts.Audience,
	}

	if opts.ProjectID != "" {
		requestToken.AdminRealmIDs = append(requestToken.AdminRealmIDs, opts.ProjectID)
		requestToken.RealmIDs = append(requestToken.RealmIDs, opts.ProjectID)
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS512, requestToken)
	return token.SignedString(t.jwtKey)
}
