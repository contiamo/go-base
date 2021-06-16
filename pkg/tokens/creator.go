package tokens

import (
	"crypto/rsa"
	"errors"
	"time"

	authz "github.com/contiamo/go-base/v4/pkg/http/middlewares/authorization"
	"github.com/dgrijalva/jwt-go"

	uuid "github.com/satori/go.uuid"
)

var (
	// ErrNoPrivateKeySpecified occurs when the private key was not set
	// and there was an attempt to create a token
	ErrNoPrivateKeySpecified = errors.New("private key is nil")
)

type token authz.Claims

// Valid implements jwt.Claims interface.
func (t token) Valid() error {
	return authz.Claims(t).Validate()
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
		IssuedAt:                       authz.FromTime(now),
		NotBefore:                      authz.FromTime(now.Add(-1 * maxSkew)),
		Expires:                        authz.FromTime(now.Add(t.lifetime)),
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

	// alternatively we could have use `claims.ToJWT`
	// because t.jwtKey is an rsa.PrivateKey, we will get the same
	// token signed using jwt.SigningMethodRS512
	// return requestToken.ToJWT(t.jwtKey)

	// wrapping authz.Claims as a jwt.Claims allows us to avoid a json.Marshal -> json.Unmarshal
	// that authz.Claims.ToJWT uses internally
	token := jwt.NewWithClaims(jwt.SigningMethodRS512, requestToken)
	return token.SignedString(t.jwtKey)
}
