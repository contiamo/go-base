package tokens

import (
	"context"
	"crypto/rsa"
	"crypto/sha512"
	"encoding/hex"
	"os"
	"path"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

var (
	// ErrKeyNotFound occurs when the key function cannot find a key in the cache
	ErrKeyNotFound = errors.New("specified key not found")
	// ErrMalformedKeyID occurs when the `KeyIDHeaderName` value in JWT header is absent or has a wrong type
	ErrMalformedKeyID = errors.New("malformed key ID in the JWT header")
	// ErrUnsupportedSigningMethod occurs when a JWT header specifies an unsupported signing method
	ErrUnsupportedSigningMethod = errors.New("signing method is not supported")
)

const (
	// KeyIDHeaderName is the expected header name in a JWT token
	KeyIDHeaderName = "kid"
)

type keyEntry struct {
	// Filename is the filename the key was loaded from
	Filename string
	// ModTime is the last modification timstamp of the file
	ModTime time.Time
	// Size is the file size
	Size int64
	// Hash is hex encoded SHA512/256 hash from the file content
	Hash string
	// Key is an RSA public key ready to be used for JWT signature validation
	Key *rsa.PublicKey
}

// PublicKeyMap defines operations on the map of public keys used for JWT validation
type PublicKeyMap interface {
	// MaintainCache runs a synchronization loop that reads the public keys directory
	// and refreshes the in-memory cache for quick access.
	MaintainCache(ctx context.Context, interval time.Duration) error
	// GetKeyFunction returns a key function that can be used in the JWT library
	GetKeyFunction() jwt.Keyfunc
}

func NewPublicKeyMap(directoryPath string) (PublicKeyMap, error) {
	m := &publicKeyMap{
		rw:            &sync.RWMutex{},
		directoryPath: directoryPath,
	}
	return m, m.init()
}

type publicKeyMap struct {
	keysByFilenames map[string]*keyEntry
	keysByHashes    map[string]*keyEntry
	directoryPath   string
	rw              *sync.RWMutex
}

func (m *publicKeyMap) MaintainCache(ctx context.Context, interval time.Duration) (err error) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for range ticker.C {
		// the only error we should expect is the context cancellation
		// the rest of the errors are just logged
		err = m.sync(ctx)
		if err != nil {
			return err
		}
	}

	return nil
}

func (m *publicKeyMap) GetKeyFunction() jwt.Keyfunc {
	return m.keyFunc
}

func (m *publicKeyMap) lookup(hash string) (entry *keyEntry, ok bool) {
	m.rw.RLock()
	entry, ok = m.keysByHashes[hash]
	m.rw.RUnlock()
	return entry, ok
}

func (m *publicKeyMap) keyFunc(token *jwt.Token) (interface{}, error) {
	_, ok := token.Method.(*jwt.SigningMethodRSA)
	if !ok {
		return nil, errors.Wrapf(
			ErrUnsupportedSigningMethod,
			"signing method: %v",
			token.Header["alg"],
		)
	}
	kid, ok := token.Header[KeyIDHeaderName].(string)
	if !ok {
		return nil, errors.Wrapf(ErrMalformedKeyID, "%s=%+v", KeyIDHeaderName, token.Header[KeyIDHeaderName])
	}

	entry, ok := m.lookup(kid)
	if !ok {
		return nil, errors.Wrapf(ErrKeyNotFound, "%s=%s", KeyIDHeaderName, kid)
	}

	return entry.Key, nil
}

func (m *publicKeyMap) init() (err error) {
	files, err := os.ReadDir(m.directoryPath)
	if err != nil {
		return errors.Wrapf(err, "failed to read directory with keys: %s", m.directoryPath)
	}

	keysByFilenames := make(map[string]*keyEntry, len(files))
	keysByHashes := make(map[string]*keyEntry, len(files))

	for _, file := range files {
		if !file.Type().IsRegular() {
			continue
		}
		key, err := m.fileToKeyEntry(file)
		if err != nil {
			logrus.
				WithField("file", file.Name()).
				WithError(err).
				Warn("failed to read the public key, skipped")
			continue
		}

		keysByFilenames[key.Filename] = key
		keysByHashes[key.Hash] = key
	}

	keysAdded := len(keysByFilenames)

	m.rw.Lock()
	m.keysByFilenames = keysByFilenames
	m.keysByHashes = keysByHashes
	m.rw.Unlock()

	logrus.
		WithField("added", keysAdded).
		Debug("public keys cache has been initialized.")

	return nil
}

func (m *publicKeyMap) clear() {
	m.rw.RLock()
	m.keysByFilenames = map[string]*keyEntry{}
	m.keysByHashes = map[string]*keyEntry{}
	m.rw.RUnlock()
}

func (m *publicKeyMap) sync(ctx context.Context) (err error) {
	// sync algorithm:

	// 1. ReadLock
	// 2. Clone `keysByFilenames` into `currentKeys`
	// 3. Unlock
	// 4. Initialize `deletes := map[string]struct{}` of known filenames (keys of `currentKeys` map)
	// 5. Define counters `keysAdded` and `keysUpdated`
	// 6. Read files (only first level) in the given directly using os.ReadDir (not recursive) on each file:
	//     1. delete(deletes, filename) – we mark a seen file, no need to delete
	//     2. for a file that has a known filename, matching modtime and size do nothing and continue to the next file. We don't check the hashes, it's too expensive to do for each file
	//     3. for a known filename but not matching properties we try load a public key and store it in `currentKeys`, increment `keysUpdated`
	//     4. for a new file try to load a public key and compute file's hash, store into `currentKeys`, increment `keysAdded`
	// 7. Delete from `currentKeys` those files that are left in `deletes` set.
	// 8. Build an updated `currentKeyHashes := map[string]*KeyEntry` a map of hashes to key entries.
	// 9. WriteLock
	// 10. Replace `keysByFilenames` and `keysByHashes` with `currentKeys` and `currentKeyHashes` respectively
	// 11. Unlock

	err = ctx.Err()
	if err != nil {
		return err
	}

	logrus.Debug("updating the public keys cache...")

	logrus.
		WithField("path", m.directoryPath).
		Debug("reading the keys directory...")

	// first try if even can read the directory
	files, err := os.ReadDir(m.directoryPath)
	if err != nil {
		logrus.
			WithField("path", m.directoryPath).
			WithError(err).
			Error("failed to read directory with keys, clearing the cache...")

		m.clear()

		logrus.
			WithField("path", m.directoryPath).
			Debug("cache cleared.")

		return nil
	}

	if len(files) == 0 {
		logrus.
			WithField("path", m.directoryPath).
			Debug("no keys have been found, clearing the cache...")

		m.clear()

		logrus.
			WithField("path", m.directoryPath).
			Debug("cache cleared.")

		return nil
	}

	logrus.
		WithField("path", m.directoryPath).
		WithField("file_count", len(files)).
		Debug("key files have been found.")

	err = ctx.Err()
	if err != nil {
		return err
	}

	// we keep the lock time very short, and we don't expect too many keys

	m.rw.RLock()

	// clone `keysByFilenames` into `currentKeys`
	// Initialize `deletes := map[string]struct{}` of known filenames
	currentKeys := make(map[string]*keyEntry, len(m.keysByFilenames))
	deletes := make(map[string]struct{}, len(m.keysByFilenames))
	for filename, entry := range m.keysByFilenames {
		currentKeys[filename] = entry
		deletes[filename] = struct{}{}
	}

	m.rw.RUnlock()

	var (
		keysAdded, keysUpdated, keysKept int
	)

	for _, file := range files {
		err = ctx.Err()
		if err != nil {
			return err
		}
		if !file.Type().IsRegular() {
			continue
		}

		filename := file.Name()

		knownKey, exists := currentKeys[filename]

		// for a file that has a known filename, matching modtime and size
		// do nothing and continue to the next file.
		// We don't check the hashes, it's too expensive to do for each file
		if exists {
			fileInfo, err := file.Info()
			if err != nil {
				logrus.
					WithField("file", file.Name()).
					WithError(err).
					Warn("failed to compare file change, skipped")
				continue
			}

			if fileInfo.ModTime() == knownKey.ModTime && fileInfo.Size() == knownKey.Size {
				// mark the key as valid, so it's not deleted later
				delete(deletes, filename)
				continue
			}
		}

		key, err := m.fileToKeyEntry(file)
		if err != nil {
			logrus.
				WithField("file", file.Name()).
				WithError(err).
				Warn("failed to read the public key, skipped")
			continue
		}
		currentKeys[filename] = key
		if exists {
			keysUpdated++
		} else {
			keysAdded++
		}
		// mark the key as valid, so it's not deleted later
		delete(deletes, filename)
	}

	if keysAdded == 0 && keysUpdated == 0 && len(deletes) == 0 {
		logrus.
			WithField("key_count", len(currentKeys)).
			Debug("no change detected, keeping the current public keys cache")
		return nil
	}

	// delete from `currentKeys` those files that are left in `deletes` set.
	for filename := range deletes {
		delete(currentKeys, filename)
	}
	keysKept = len(currentKeys) - keysAdded - keysUpdated - len(deletes)

	currentKeyHashes := make(map[string]*keyEntry, len(currentKeys))
	for _, key := range currentKeys {
		currentKeyHashes[key.Hash] = key
	}

	m.rw.Lock()
	m.keysByFilenames = currentKeys
	m.keysByHashes = currentKeyHashes
	m.rw.Unlock()

	logrus.
		WithField("added", keysAdded).
		WithField("updated", keysUpdated).
		WithField("deleted", len(deletes)).
		WithField("kept", keysKept).
		Debug("public keys cache has been updated.")

	return nil
}

func (m *publicKeyMap) fileToKeyEntry(file os.DirEntry) (key *keyEntry, err error) {
	fileInfo, err := file.Info()
	if err != nil {
		return nil, err
	}

	filename := file.Name()
	fullPath := path.Join(m.directoryPath, filename)
	bytes, err := os.ReadFile(fullPath)
	if err != nil {
		return nil, err
	}

	rsaKey, err := jwt.ParseRSAPublicKeyFromPEM(bytes)
	if err != nil {
		return nil, err
	}

	hashBytes := sha512.Sum512_256(bytes)
	hash := hex.EncodeToString(hashBytes[:])

	return &keyEntry{
		Filename: filename,
		ModTime:  fileInfo.ModTime(),
		Size:     fileInfo.Size(),
		Hash:     hash,
		Key:      rsaKey,
	}, nil
}
