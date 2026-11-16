package password

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"golang.org/x/crypto/argon2"
	"strings"
)

const prefix = "$argon2id$v=19$m=19456,t=2,p=1$"

var work = make(chan struct{}, 2)

func derive(ctx context.Context, value string, salt []byte) ([]byte, error) {
	select {
	case work <- struct{}{}:
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	defer func() { <-work }()
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	key := argon2.IDKey([]byte(value), salt, 2, 19456, 1, 32)
	return key, ctx.Err()
}
func Hash(value string) (string, error) { return HashContext(context.Background(), value) }
func HashContext(ctx context.Context, value string) (string, error) {
	if len(value) < 1 || len(value) > 128 || strings.ContainsRune(value, 0) {
		return "", errors.New("password must contain 1..128 bytes without NUL")
	}
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	key, err := derive(ctx, value, salt)
	if err != nil {
		return "", err
	}
	return prefix + base64.RawStdEncoding.EncodeToString(salt) + "$" + base64.RawStdEncoding.EncodeToString(key), nil
}
func Verify(stored, value string) (bool, bool) {
	ok, legacy, _ := VerifyContext(context.Background(), stored, value)
	return ok, legacy
}

// Dummy performs the same bounded KDF work for an unknown account. This reduces
// user-enumeration timing differences; network/database timing is not constant.
func Dummy(ctx context.Context, value string) error {
	_, err := derive(ctx, value, make([]byte, 16))
	return err
}
func VerifyContext(ctx context.Context, stored, value string) (bool, bool, error) {
	if !strings.HasPrefix(stored, "$") {
		matches := subtle.ConstantTimeCompare([]byte(stored), []byte(value)) == 1
		return matches, true, Dummy(ctx, value)
	}
	invalid := func() (bool, bool, error) { return false, false, Dummy(ctx, value) }
	if !strings.HasPrefix(stored, prefix) {
		return invalid()
	}
	parts := strings.Split(strings.TrimPrefix(stored, prefix), "$")
	if len(parts) != 2 {
		return invalid()
	}
	salt, e := base64.RawStdEncoding.DecodeString(parts[0])
	if e != nil || len(salt) != 16 {
		return invalid()
	}
	expected, e := base64.RawStdEncoding.DecodeString(parts[1])
	if e != nil || len(expected) != 32 {
		return invalid()
	}
	actual, err := derive(ctx, value, salt)
	if err != nil {
		return false, false, err
	}
	return subtle.ConstantTimeCompare(actual, expected) == 1, false, nil
}
