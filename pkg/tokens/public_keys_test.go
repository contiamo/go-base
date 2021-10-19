package tokens

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha512"
	"crypto/x509"
	"encoding/hex"
	"encoding/pem"
	"io/ioutil"
	"os"
	"path"
	"testing"
	"time"

	ccrypto "github.com/contiamo/go-base/v4/pkg/crypto"
	"github.com/golang-jwt/jwt/v4"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

func TestPublicKeysGetKeyFunction(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	logrus.SetOutput(ioutil.Discard)
	defer logrus.SetOutput(os.Stdout)

	t.Run("returns error if the path is empty", func(t *testing.T) {
		_, err := NewPublicKeyMap("")
		require.Error(t, err)
	})

	t.Run("returns error if the path is wrong", func(t *testing.T) {
		_, err := NewPublicKeyMap("/definitely-not-existing-path")
		require.Error(t, err)
	})

	t.Run("returns no error on empty dir", func(t *testing.T) {
		tmpPath, err := ioutil.TempDir(os.TempDir(), "public-keys-test")
		require.NoError(t, err)
		defer func() {
			_ = os.RemoveAll(tmpPath)
		}()

		_, err = NewPublicKeyMap(tmpPath)
		require.NoError(t, err)
	})

	t.Run("reads the key on init", func(t *testing.T) {
		tmpPath, err := ioutil.TempDir(os.TempDir(), "public-keys-test")
		require.NoError(t, err)
		defer func() {
			_ = os.RemoveAll(tmpPath)
		}()

		kid := createPublicKey(t, tmpPath)
		cache, err := NewPublicKeyMap(tmpPath)
		require.NoError(t, err)

		key, err := cache.KeyFunction(createToken(kid))
		require.NoError(t, err)
		require.NotNil(t, key)
	})

	t.Run("adds the key when the file gets added", func(t *testing.T) {
		tmpPath, err := ioutil.TempDir(os.TempDir(), "public-keys-test")
		require.NoError(t, err)
		defer func() {
			_ = os.RemoveAll(tmpPath)
		}()
		ctx, cancel := context.WithCancel(ctx)
		defer cancel() // so we don't have errors after removing the directory

		cache, err := NewPublicKeyMap(tmpPath)
		require.NoError(t, err)

		go func() {
			_ = cache.MaintainCache(ctx, 10*time.Millisecond)
		}()

		kid := createPublicKey(t, tmpPath)

		<-time.After(20 * time.Millisecond)

		key, err := cache.KeyFunction(createToken(kid))
		require.NoError(t, err)
		require.NotNil(t, key)
	})

	t.Run("updates the key if the file gets re-written", func(t *testing.T) {
		tmpPath, err := ioutil.TempDir(os.TempDir(), "public-keys-test")
		require.NoError(t, err)
		defer func() {
			_ = os.RemoveAll(tmpPath)
		}()
		ctx, cancel := context.WithCancel(ctx)
		defer cancel() // so we don't have errors after removing the directory

		filename, err := ccrypto.GenerateRandomString(8)
		require.NoError(t, err)

		kid1 := createPublicKeyWithName(t, tmpPath, filename)

		cache, err := NewPublicKeyMap(tmpPath)
		require.NoError(t, err)

		go func() {
			_ = cache.MaintainCache(ctx, 10*time.Millisecond)
		}()

		// override the file with a new key
		kid2 := createPublicKeyWithName(t, tmpPath, filename)

		<-time.After(20 * time.Millisecond)

		_, err = cache.KeyFunction(createToken(kid1))
		require.ErrorIs(t, err, ErrKeyNotFound)

		key, err := cache.KeyFunction(createToken(kid2))
		require.NoError(t, err)
		require.NotNil(t, key)
	})

	t.Run("deletes the key if the file gets deleted", func(t *testing.T) {
		tmpPath, err := ioutil.TempDir(os.TempDir(), "public-keys-test")
		require.NoError(t, err)
		defer func() {
			_ = os.RemoveAll(tmpPath)
		}()
		ctx, cancel := context.WithCancel(ctx)
		defer cancel() // so we don't have errors after removing the directory

		filename, err := ccrypto.GenerateRandomString(8)
		require.NoError(t, err)
		_ = createPublicKey(t, tmpPath)

		kid := createPublicKeyWithName(t, tmpPath, filename)

		cache, err := NewPublicKeyMap(tmpPath)
		require.NoError(t, err)

		go func() {
			_ = cache.MaintainCache(ctx, 10*time.Millisecond)
		}()

		key, err := cache.KeyFunction(createToken(kid))
		require.NoError(t, err)
		require.NotNil(t, key)

		err = os.Remove(path.Join(tmpPath, filename))
		require.NoError(t, err)

		<-time.After(20 * time.Millisecond)

		_, err = cache.KeyFunction(createToken(kid))
		require.ErrorIs(t, err, ErrKeyNotFound)
	})

	t.Run("clears the cache when there are no keys anymore", func(t *testing.T) {
		tmpPath, err := ioutil.TempDir(os.TempDir(), "public-keys-test")
		require.NoError(t, err)
		defer func() {
			_ = os.RemoveAll(tmpPath)
		}()
		ctx, cancel := context.WithCancel(ctx)
		defer cancel() // so we don't have errors after removing the directory

		filename, err := ccrypto.GenerateRandomString(8)
		require.NoError(t, err)

		kid := createPublicKeyWithName(t, tmpPath, filename)

		cache, err := NewPublicKeyMap(tmpPath)
		require.NoError(t, err)

		go func() {
			_ = cache.MaintainCache(ctx, 10*time.Millisecond)
		}()

		key, err := cache.KeyFunction(createToken(kid))
		require.NoError(t, err)
		require.NotNil(t, key)

		// removing the only file
		err = os.Remove(path.Join(tmpPath, filename))
		require.NoError(t, err)

		<-time.After(20 * time.Millisecond)

		_, err = cache.KeyFunction(createToken(kid))
		require.ErrorIs(t, err, ErrKeyNotFound)
	})

	t.Run("keeps the key if there is no change", func(t *testing.T) {
		tmpPath, err := ioutil.TempDir(os.TempDir(), "public-keys-test")
		require.NoError(t, err)
		defer func() {
			_ = os.RemoveAll(tmpPath)
		}()
		ctx, cancel := context.WithCancel(ctx)
		defer cancel() // so we don't have errors after removing the directory

		kid := createPublicKey(t, tmpPath)

		cache, err := NewPublicKeyMap(tmpPath)
		require.NoError(t, err)

		go func() {
			_ = cache.MaintainCache(ctx, 10*time.Millisecond)
		}()

		key, err := cache.KeyFunction(createToken(kid))
		require.NoError(t, err)
		require.NotNil(t, key)

		<-time.After(20 * time.Millisecond)

		key, err = cache.KeyFunction(createToken(kid))
		require.NoError(t, err)
		require.NotNil(t, key)
	})

	t.Run("does not corrupt the cache when there is an invalid key file", func(t *testing.T) {
		tmpPath, err := ioutil.TempDir(os.TempDir(), "public-keys-test")
		require.NoError(t, err)
		defer func() {
			_ = os.RemoveAll(tmpPath)
		}()
		ctx, cancel := context.WithCancel(ctx)
		defer cancel() // so we don't have errors after removing the directory

		kid := createPublicKey(t, tmpPath)

		cache, err := NewPublicKeyMap(tmpPath)
		require.NoError(t, err)

		go func() {
			_ = cache.MaintainCache(ctx, 10*time.Millisecond)
		}()

		key, err := cache.KeyFunction(createToken(kid))
		require.NoError(t, err)
		require.NotNil(t, key)

		err = os.WriteFile(path.Join(tmpPath, "invalid"), []byte("invalid"), os.ModePerm)
		require.NoError(t, err)

		<-time.After(20 * time.Millisecond)

		key, err = cache.KeyFunction(createToken(kid))
		require.NoError(t, err)
		require.NotNil(t, key)
	})

	t.Run("key function returns expected errors", func(t *testing.T) {
		tmpPath, err := ioutil.TempDir(os.TempDir(), "public-keys-test")
		require.NoError(t, err)
		defer func() {
			_ = os.RemoveAll(tmpPath)
		}()

		cache, err := NewPublicKeyMap(tmpPath)
		require.NoError(t, err)

		t.Run("ErrUnsupportedSigningMethod", func(t *testing.T) {
			_, err = cache.KeyFunction(&jwt.Token{})
			require.ErrorIs(t, err, ErrUnsupportedSigningMethod)
		})
		t.Run("ErrMalformedKeyID", func(t *testing.T) {
			token := &jwt.Token{
				Method: &jwt.SigningMethodRSA{},
			}
			_, err = cache.KeyFunction(token)
			require.ErrorIs(t, err, ErrMalformedKeyID)
		})
		t.Run("ErrKeyNotFound", func(t *testing.T) {
			token := &jwt.Token{
				Header: map[string]interface{}{
					KeyIDHeaderName: "invalid",
				},
				Method: &jwt.SigningMethodRSA{},
			}
			_, err = cache.KeyFunction(token)
			require.ErrorIs(t, err, ErrKeyNotFound)
		})
	})

	t.Run("key function satisfies the jwt.KeyFunc type", func(t *testing.T) {
		tmpPath, err := ioutil.TempDir(os.TempDir(), "public-keys-test")
		require.NoError(t, err)
		defer func() {
			_ = os.RemoveAll(tmpPath)
		}()

		cache, err := NewPublicKeyMap(tmpPath)
		require.NoError(t, err)

		_, _ = jwt.Parse("", cache.KeyFunction)
	})
}

func createToken(kid string) *jwt.Token {
	return &jwt.Token{
		Header: map[string]interface{}{
			KeyIDHeaderName: kid,
		},
		Method: &jwt.SigningMethodRSA{},
	}
}

func createPublicKeyWithName(t *testing.T, dir, filename string) (kid string) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	publicKeyBytes, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
	require.NoError(t, err)

	buffer := bytes.NewBuffer(nil)
	publicKeyBlock := &pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: publicKeyBytes,
	}
	err = pem.Encode(buffer, publicKeyBlock)
	require.NoError(t, err)

	pemBytes := buffer.Bytes()
	hashBytes := sha512.Sum512_256(pemBytes)
	kid = hex.EncodeToString(hashBytes[:])

	err = os.WriteFile(path.Join(dir, filename), pemBytes, os.ModePerm)
	require.NoError(t, err)

	return kid
}

func createPublicKey(t *testing.T, dir string) (kid string) {
	filename, err := ccrypto.GenerateRandomString(8)
	require.NoError(t, err)

	return createPublicKeyWithName(t, dir, filename)
}
