// Package password stores versioned Argon2id credentials. Legacy plaintext is
// accepted only to upgrade existing development accounts after authentication.
package password

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"golang.org/x/crypto/argon2"
	"strings"
)

const prefix = "$argon2id$v=19$m=19456,t=2,p=1$"

var work = make(chan struct{}, 2) // at most 38 MiB of hashing memory per process

func derive(value string, salt []byte) []byte {
	work <- struct{}{}
	defer func() { <-work }()
	return argon2.IDKey([]byte(value), salt, 2, 19456, 1, 32)
}
func Hash(value string) (string, error) {
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	return prefix + base64.RawStdEncoding.EncodeToString(salt) + "$" + base64.RawStdEncoding.EncodeToString(derive(value, salt)), nil
}

// Verify returns whether the credential matches and needs a legacy upgrade.
// Unknown hash formats fail closed rather than selecting unbounded parameters.
func Verify(stored, value string) (bool, bool) {
	if !strings.HasPrefix(stored, "$") {
		return subtle.ConstantTimeCompare([]byte(stored), []byte(value)) == 1, true
	}
	if !strings.HasPrefix(stored, prefix) {
		return false, false
	}
	parts := strings.Split(strings.TrimPrefix(stored, prefix), "$")
	if len(parts) != 2 {
		return false, false
	}
	salt, e := base64.RawStdEncoding.DecodeString(parts[0])
	if e != nil || len(salt) != 16 {
		return false, false
	}
	expected, e := base64.RawStdEncoding.DecodeString(parts[1])
	if e != nil || len(expected) != 32 {
		return false, false
	}
	return subtle.ConstantTimeCompare(derive(value, salt), expected) == 1, false
}
