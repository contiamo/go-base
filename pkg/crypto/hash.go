package crypto

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"hash"
	"io"
	"strings"

	"golang.org/x/crypto/sha3"
)

var (
	// DefaultHasher is the default implementation for hashing things
	// It outputs 32 Bytes and uses a SHA3-256 hash in the current configuration.
	// Its generic security strength is 256 bits against preimage attacks,
	// and 128 bits against collision attacks.
	defaultHasher = basicHasher{sha3.New256()}
)

// Hash is a convenience function calling the default hasher
// WARNING: only pass in data that is json-marshalable. If not, the worst case scenario is that you passed in data with circular references and this will just blow up your CPU
func Hash(data ...interface{}) ([]byte, error) {
	return defaultHasher.Hash(data...)
}

// HashToString is a convenience function calling the default hasher and encoding the result as hex string
func HashToString(data ...interface{}) (string, error) {
	hash, err := defaultHasher.Hash(data...)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(hash), nil
}

// Hasher provides a method for hashing arbitrary data types
type Hasher interface {
	Hash(data ...interface{}) ([]byte, error)
}

type basicHasher struct {
	hash hash.Hash
}

func (h basicHasher) Hash(args ...interface{}) ([]byte, error) {
	h.hash.Reset()

	for _, data := range args {
		var (
			reader       io.Reader
			encoderError error
		)

		// setup reader for the data
		switch d := data.(type) {
		case io.Reader:
			reader = d
		case []byte:
			reader = bytes.NewReader(d)
		case string:
			reader = strings.NewReader(d)
		case fmt.Stringer:
			reader = strings.NewReader(d.String())
		default:
			r, w := io.Pipe()
			encoder := json.NewEncoder(w)
			go func() {
				defer w.Close()
				encoderError = encoder.Encode(data)
			}()
			reader = r
		}

		// hash all the data
		_, err := io.Copy(h.hash, reader)
		if err != nil {
			return nil, err
		}

		// check encoder error
		if encoderError != nil {
			return nil, encoderError
		}
	}

	return h.hash.Sum(nil), nil
}
