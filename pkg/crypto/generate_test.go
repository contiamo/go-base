package crypto

import (
	"bytes"
	"testing"
	"time"
)

func TestRandomString(t *testing.T) {
	t.Run("does not repeat the same value twice", func(t *testing.T) {
		r1, err := GenerateRandomString(16)
		if err != nil {
			t.Fatal(err)
		}
		if len(r1) != 16 {
			t.Fatal("string length must be 15 characters")
		}

		r2, err := GenerateRandomString(16)
		if err != nil {
			t.Fatal(err)
		}
		if len(r2) != 16 {
			t.Fatal("string length must be 15 characters")
		}

		if r1 == r2 {
			t.Fatal("must produce different strings in a row")
		}
	})

	t.Run("supports lengths %2 != 0", func(t *testing.T) {
		r1, err := GenerateRandomString(15)
		if err != nil {
			t.Fatal(err)
		}
		if len(r1) != 15 {
			t.Fatal("string length must be 15 characters")
		}
	})
}

func TestGenerateDecodeAndVerifyToken(t *testing.T) {
	key := PassphraseToKey("some very secure passphrase no hacker can hack")
	text := []byte("some very secret text to encrypt")

	t.Run("preserves the data", func(t *testing.T) {
		token, err := GenerateToken(text, key)
		if err != nil {
			t.Fatal(err)
		}

		data, err := DecodeAndVerifyToken(token, key, time.Hour)
		if err != nil {
			t.Fatal(err)
		}

		if !bytes.Equal(text, data) {
			t.Fatal("data does not match with the initial text")
		}
	})

	t.Run("expires", func(t *testing.T) {
		token, err := GenerateToken(text, key)
		if err != nil {
			t.Fatal(err)
		}

		<-time.After(2 * time.Second)

		data, err := DecodeAndVerifyToken(token, key, time.Second)
		if err != ErrTokenExpired {
			t.Fatal(err)
		}
		if len(data) != 0 {
			t.Fatal("data should be empty")
		}
	})
}
