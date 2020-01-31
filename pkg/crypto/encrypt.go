package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha512"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"io"
)

var (
	// ErrCipherTooShort occurs when `Decrypt` does not
	// have input of enough length to decrypt using AES256
	ErrCipherTooShort = errors.New("crypto: cipher plainText is too short for AES encryption")
	// ErrCorruptedMessage occurs when an attempt of unsealing a message
	// does not pass the authentication check
	ErrCorruptedMessage = errors.New("crypto: the message didn't pass the authentication check")
)

// PassphraseToKey converts a string to a key for encryption.
//
// This function must be used STRICTLY ONLY for generating
// an encryption key out of a passphrase.
// Please don't use this function for hashing user-provided values.
// It uses SHA2 for simplicity and it's faster but less secure than SHA3.
// User-provided data should use SHA3 or bcrypt.
func PassphraseToKey(passphrase string) (key []byte) {
	// SHA512/256 will return exactly 32 bytes which is exactly
	// the length of the key needed for AES256 encryption
	hash := sha512.Sum512_256([]byte(passphrase))
	return hash[:]
}

// Encrypt encrypts content with a key using AES256 CFB mode
func Encrypt(plainText, key []byte) (cipherText []byte, err error) {
	// code is taken from here https://golang.org/pkg/crypto/cipher/#NewCFBEncrypter
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	cipherText = make([]byte, aes.BlockSize+len(plainText))
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

// Decrypt decrypts content with a key using AES256 CFB mode
func Decrypt(cipherText, key []byte) (plainText []byte, err error) {
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

	plainText = make([]byte, len(cipherText))
	stream.XORKeyStream(plainText, cipherText)

	return plainText, nil
}

// DecryptFromString decrypts a string with a key
func DecryptFromString(cipherTextStr string, key []byte) ([]byte, error) {
	cipherText, err := hex.DecodeString(cipherTextStr)
	if err != nil {
		return nil, err
	}
	return Decrypt(cipherText, key)
}

// Seal implements authenticated encryption using the MAC-then-Encrypt (MtE) approach.
// It's using SHA3-256 for MAC and AES256 CFB for encryption.
// https://en.wikipedia.org/wiki/Authenticated_encryption#MAC-then-Encrypt_(MtE)
func Seal(plainText, key []byte) (cipherText []byte, err error) {
	mac := hmac.New(sha512.New512_256, key)

	// the doc says it never returns an error, but we don't trust it
	_, err = mac.Write(plainText)
	if err != nil {
		return nil, err
	}
	messageMAC := mac.Sum(nil)
	messageAndMAC := append(plainText, messageMAC...)

	return Encrypt(messageAndMAC, key)
}

// SealToString runs `Seal` and then encodes the result into base64.
func SealToString(plainText, key []byte) (string, error) {
	bytes, err := Seal(plainText, key)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(bytes), nil
}

// Unseal decrypts and authenticates the data encrypted by Seal
func Unseal(cipherText, key []byte) (plainText []byte, err error) {
	messageAndMAC, err := Decrypt(cipherText, key)
	if err != nil {
		return nil, err
	}

	splitPoint := len(messageAndMAC) - sha512.Size256
	messageMAC := messageAndMAC[splitPoint:]
	plainText = messageAndMAC[:splitPoint]

	mac := hmac.New(sha512.New512_256, key)
	mac.Write(plainText)
	expectedMAC := mac.Sum(nil)

	if !hmac.Equal(expectedMAC, messageMAC) {
		return nil, ErrCorruptedMessage
	}

	return plainText, err
}

// UnsealFromString decodes from Base64 and applies `Unseal`.
func UnsealFromString(cipherTextStr string, key []byte) ([]byte, error) {
	cipherText, err := base64.StdEncoding.DecodeString(cipherTextStr)
	if err != nil {
		return nil, err
	}
	return Unseal(cipherText, key)
}
