package config

import (
	"crypto/rsa"
	"io/ioutil"

	jwt "github.com/dgrijalva/jwt-go"
	"github.com/pkg/errors"
)

// JWT contains all JWT related parameters
type JWT struct {
	// PublicKeyPath is a path to the public key for JWT signature verification
	PublicKeyPath string `json:"publicKeyPath"`
	// PrivateKeyPath is a path to the private key for signing JWT
	PrivateKeyPath string `json:"privateKeyPath"`
}

// GetPublicKey gets the encryption key from a given path
func (j *JWT) GetPublicKey() (publicKey *rsa.PublicKey, err error) {
	if j.PublicKeyPath == "" {
		return nil, errors.New("path to the public key file is empty")
	}

	keyBytes, err := ioutil.ReadFile(j.PublicKeyPath)
	if err != nil {
		return nil, errors.Wrapf(err, "can not read the public key file `%s`", j.PublicKeyPath)
	}

	return jwt.ParseRSAPublicKeyFromPEM(keyBytes)
}

// GetPrivateKey gets the encryption key from a given path
func (j *JWT) GetPrivateKey() (privateKey *rsa.PrivateKey, err error) {
	if j.PrivateKeyPath == "" {
		return nil, errors.New("path to the private key file is empty")
	}

	keyBytes, err := ioutil.ReadFile(j.PrivateKeyPath)
	if err != nil {
		return nil, errors.Wrapf(err, "can not read the private key file `%s`", j.PrivateKeyPath)
	}

	return jwt.ParseRSAPrivateKeyFromPEM(keyBytes)
}
