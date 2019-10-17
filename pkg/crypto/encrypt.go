package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha512"
	"encoding/hex"
	"errors"
	"io"
)

var (
	// ErrCipherTooShort occurs when `Decrypt` does not
	// have input of enough length to decrypt using AES256
	ErrCipherTooShort = errors.New("crypto: cipher plainText is too short for AES encryption")
)

// PassphraseToKey converts a string to a key for encryption.
//
// This function must be used STRICTLY ONLY for generating
// an encryption key out of a passphrase.
// Please don't use this function for hashing user-provided values.
// It uses SHA2 for simplicity but it's slower. User-provided data should use SHA3
// because of its better performance.
func PassphraseToKey(passphrase string) (key []byte) {
	// SHA512/256 will return exactly 32 bytes which is exactly
	// the length of the key needed for AES256 encryption
	hash := sha512.Sum512_256([]byte(passphrase))
	return hash[:]
}

// Encrypt encrypts content with a key using AES256
func Encrypt(plainText, key []byte) (encrypted []byte, err error) {
	// code is taken from here https://golang.org/pkg/crypto/cipher/#NewCFBEncrypter
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	cipherText := make([]byte, aes.BlockSize+len(plainText))
	iv := cipherText[:aes.BlockSize]
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return nil, err
	}

	stream := cipher.NewCFBEncrypter(block, iv)
	stream.XORKeyStream(cipherText[aes.BlockSize:], plainText)

	return cipherText, nil
}

// EncryptToString encrypts content with a key using AES256
// and encodes it to a hexadecimal string
func EncryptToString(plainText, key []byte) (string, error) {
	bytes, err := Encrypt(plainText, key)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// Decrypt decrypts content with a key using AES256
func Decrypt(cipherText, key []byte) (decrypted []byte, err error) {
	// code is taken from here https://golang.org/pkg/crypto/cipher/#NewCFBDecrypter
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	if len(cipherText) < aes.BlockSize {
		return nil, ErrCipherTooShort
	}
	iv := cipherText[:aes.BlockSize]
	cipherText = cipherText[aes.BlockSize:]

	stream := cipher.NewCFBDecrypter(block, iv)

	plainText := make([]byte, len(cipherText))
	stream.XORKeyStream(plainText, cipherText)

	return plainText, nil
}

// DecryptFromString decrypts a string with a key
func DecryptFromString(cipherTextStr string, key []byte) (decrypted []byte, err error) {
	cipherText, err := hex.DecodeString(cipherTextStr)
	if err != nil {
		return nil, err
	}
	return Decrypt(cipherText, key)
}
