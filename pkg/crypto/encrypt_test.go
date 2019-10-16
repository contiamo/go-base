package crypto

import (
	"bytes"
	"testing"
)

func TestPassphraseToKey(t *testing.T) {
	passphrase := "somekey"
	key := PassphraseToKey(passphrase)
	if len(key) != 32 {
		t.Fatal("the key length must be 32 bytes")
	}
	key2 := PassphraseToKey(passphrase)
	if !bytes.Equal(key, key2) {
		t.Fatal("`PassphraseToKey` must be determenistic and always return the same value for the same parameter")
	}
}

func Test_EncryptDecrypt(t *testing.T) {
	key := PassphraseToKey("some very secure passphrase no hacker can hack")
	text := []byte("some very secret text to encrypt")

	t.Run("encrypts/decrypts bytes", func(t *testing.T) {
		cipherText, err := Encrypt(text, key)
		if err != nil {
			t.Fatal(err)
		}
		if bytes.Equal(text, cipherText) {
			t.Fatal("cipher text must differ the original text")
		}

		plainText, err := Decrypt(cipherText, key)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(text, plainText) {
			t.Fatal("decrypted text must match the original text")
		}
	})

	t.Run("encrypts/decrypts a string", func(t *testing.T) {
		cipherTextString, err := EncryptToString(text, key)
		if err != nil {
			t.Fatal(err)
		}
		if string(text) == cipherTextString {
			t.Fatal("cipher text must differ the original text")
		}

		plainText, err := DecryptFromString(cipherTextString, key)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(text, plainText) {
			t.Fatal("decrypted text must match the original text")
		}
	})

	t.Run("returns error if the cipher text is too short", func(t *testing.T) {
		_, err := Decrypt([]byte{0}, key)
		if err != ErrCipherTooShort {
			t.Fatal(err, "expected `ErrCipherTooShort`")
		}
	})
}
