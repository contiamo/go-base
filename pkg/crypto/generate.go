package crypto

import (
	"bytes"
	"crypto/rand"
	"encoding/gob"
	"encoding/hex"
	"errors"
	"io"
	"time"
)

var (
	// ErrTokenExpired occurs when the token lifetime is exceeded
	ErrTokenExpired = errors.New("crypto: token expired")
)

type token struct {
	Data      []byte
	CreatedAt time.Time
}

// GenerateRandomString generates a random string with a given length
func GenerateRandomString(length int) (string, error) {
	size := length
	// we generate bytes and it's 2 char per byte in a string
	// so we have to generate more and then trim
	if size%2 != 0 {
		size++
	}
	bytes := make([]byte, size)
	if _, err := io.ReadFull(rand.Reader, bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes)[0:length], nil
}

// GenerateToken generates a sealed token with a given ID and timestamp for
// future verification.
func GenerateToken(data, key []byte) (tokenStr string, err error) {
	t := token{
		Data:      data,
		CreatedAt: time.Now(),
	}
	b := bytes.NewBuffer(nil)
	enc := gob.NewEncoder(b)
	err = enc.Encode(t)
	if err != nil {
		return "", err
	}

	return SealToString(b.Bytes(), key)
}

// DecodeAndVerify unseals the token and verifies its lifetime
func DecodeAndVerifyToken(tokenStr string, key []byte, lifetime time.Duration) (data []byte, err error) {
	plainText, err := UnsealFromString(tokenStr, key)
	if err != nil {
		return nil, err
	}
	b := bytes.NewBuffer(plainText)
	dec := gob.NewDecoder(b)

	var t token
	err = dec.Decode(&t)
	if err != nil {
		return nil, err
	}

	if t.CreatedAt.Add(lifetime).Before(time.Now()) {
		return nil, ErrTokenExpired
	}

	return t.Data, nil
}
