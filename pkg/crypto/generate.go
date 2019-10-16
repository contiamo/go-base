package crypto

import (
	"crypto/rand"
	"encoding/hex"
	"io"
)

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
