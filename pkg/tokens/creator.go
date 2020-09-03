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

	UserID   string `json:"sub"`
	UserName string `json:"name"`
	Email    string `json:"email"`

	TenantID                       string   `json:"tenantID"`
	RealmIDs                       []string `json:"realmIDs"`
	GroupIDs                       []string `json:"groupIDs"`
	AllowedIPs                     []string `json:"allowedIPs"`
	IsTenantAdmin                  bool     `json:"isTenantAdmin"`
	AdminRealmIDs                  []string `json:"adminRealmIDs"`
	AuthenticationMethodReferences []string `json:"amr"`
}

func (token) Valid() error {
	return nil
}

// Creator creates all kinds of signed tokens for the background tasks
type Creator interface {
	// CreateProjectAdmin creates a signed token that has admin permissions for the project.
	// Reference is passed as the JWT `amr` value
	CreateProjectAdmin(projectID string, reference string) (string, error)
	// Create creates a signed token that has no admin permissions.
	Create(reference string) (string, error)
}

// NewCreator creates a new token creator for tasks
func NewCreator(privateKey *rsa.PrivateKey, lifetime time.Duration) Creator {
	if lifetime < 5 {
		lifetime = 5 * time.Second
	}
	return &tokenCreator{
		jwtKey:   privateKey,
		lifetime: lifetime,
	}
}

type tokenCreator struct {
	jwtKey   *rsa.PrivateKey
	lifetime time.Duration
}

func (t *tokenCreator) CreateProjectAdmin(projectID string, reference string) (string, error) {
	return t.create([]string{projectID}, []string{projectID}, reference)
}
func (t *tokenCreator) Create(reference string) (string, error) {
	return t.create([]string{}, []string{}, reference)
}

func (t *tokenCreator) create(adminProjects, memberProjects []string, reference string) (string, error) {
	if t.jwtKey == nil {
		return "", ErrNoPrivateKeySpecified
	}

	now := time.Now()
	maxSkew := 5 * time.Second
	requestToken := token{
		ID:                             uuid.NewV4().String(),
		Issuer:                         "hub",
		IssuedAt:                       float64(now.Unix()),
		NotBefore:                      float64(now.Add(-1 * maxSkew).Unix()),
		Expires:                        float64(now.Add(t.lifetime).Unix()),
		UserID:                         uuid.Nil.String(),
		UserName:                       "@hub",
		Email:                          "hub@contiamo.com",
		RealmIDs:                       memberProjects,
		GroupIDs:                       []string{},
		AllowedIPs:                     []string{},
		AdminRealmIDs:                  adminProjects,
		AuthenticationMethodReferences: []string{reference},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS512, requestToken)
	return token.SignedString(t.jwtKey)
}
