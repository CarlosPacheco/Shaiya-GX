// Package session hands an authenticated login over to the world server.
// Login writes an atomic SQLite ticket; the client then connects to
// world and presents the same identity token in its 0xA301 handshake.
package session

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"database/sql"
	"encoding/binary"
	"errors"
	"time"

	"github.com/Spelunkern/Shaiya-GX/server/internal/db"
)

// MaxAge bounds how long a handoff stays valid after login.
const MaxAge = 10 * time.Minute

type Handoff struct {
	Identity          [16]byte
	AESMaterial       [32]byte // login AES key (16) + IV (16)
	UserUID           uint32
	SessionID         int64
	CredentialVersion int64
}

// NewIdentity returns a fresh random token for one login.
func NewIdentity() [16]byte {
	var id [16]byte
	if _, err := rand.Read(id[:]); err != nil {
		panic(err)
	}
	return id
}

// NewSessionID returns the bigint session id passed to the login procedures.
func NewSessionID() int64 {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		panic(err)
	}
	return int64(binary.LittleEndian.Uint64(b[:]) & 0x7FFF_FFFF_FFFF_FFFF)
}

var ErrInvalid = errors.New("no valid login handoff")

// Save replaces the previous login ticket in the configured runtime database.
func Save(h Handoff) error {
	pool, err := db.Pool(db.UserData)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), db.Timeout)
	defer cancel()
	token := sha256.Sum256(h.Identity[:])
	result, err := pool.ExecContext(ctx, `INSERT INTO GX_SessionHandoff(UserUID,Identity,AESMaterial,SessionID,ExpiresAt,CredentialVersion)
		SELECT ?1,?2,?3,?4,?5,Version
		FROM (SELECT COALESCE((SELECT Version FROM GX_AccountSecurity WHERE UserUID=?1),1) AS Version)
		WHERE ?6=0 OR ?6=Version
		ON CONFLICT(UserUID) DO UPDATE SET Identity=excluded.Identity,AESMaterial=excluded.AESMaterial,
		SessionID=excluded.SessionID,ExpiresAt=excluded.ExpiresAt,CredentialVersion=excluded.CredentialVersion`,
		h.UserUID, token[:], h.AESMaterial[:], h.SessionID, time.Now().Add(MaxAge).Unix(), h.CredentialVersion)
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err == nil && affected != 1 {
		return ErrInvalid
	}
	return err
}

// Consume first checks the token on a read-only connection. A valid candidate
// is then consumed by one guarded DELETE that rechecks expiry and revocation.
func Consume(userUID uint32, identity [16]byte) (Handoff, error) {
	reader, err := db.ReadPool()
	if err != nil {
		return Handoff{}, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), db.Timeout)
	defer cancel()
	var stored []byte
	err = reader.QueryRowContext(ctx, `SELECT Identity FROM GX_SessionHandoff WHERE UserUID=? AND ExpiresAt>unixepoch()`, userUID).Scan(&stored)
	if errors.Is(err, sql.ErrNoRows) {
		return Handoff{}, ErrInvalid
	}
	if err != nil {
		return Handoff{}, err
	}
	token := sha256.Sum256(identity[:])
	expected := token[:]
	// Tickets pending during an upgrade retain their original 16-byte encoding
	// until consumed or expired.
	if len(stored) == len(identity) {
		expected = identity[:]
	}
	if subtle.ConstantTimeCompare(stored, expected) != 1 {
		return Handoff{}, ErrInvalid
	}
	writer, err := db.Pool(db.UserData)
	if err != nil {
		return Handoff{}, err
	}
	h := Handoff{UserUID: userUID, Identity: identity}
	var material []byte
	err = writer.QueryRowContext(ctx, `DELETE FROM GX_SessionHandoff
		WHERE UserUID=?1 AND Identity=?2 AND ExpiresAt>unixepoch() AND length(AESMaterial)=32
		AND CredentialVersion=COALESCE((SELECT Version FROM GX_AccountSecurity WHERE UserUID=?1),1)
		RETURNING AESMaterial,SessionID,CredentialVersion`, userUID, stored).Scan(&material, &h.SessionID, &h.CredentialVersion)
	if errors.Is(err, sql.ErrNoRows) {
		return Handoff{}, ErrInvalid
	}
	if err != nil {
		return Handoff{}, err
	}
	copy(h.AESMaterial[:], material)
	return h, nil
}

// PurgeExpired drops old key material in bounded batches.
func PurgeExpired() error {
	pool, err := db.Pool(db.UserData)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), db.Timeout)
	defer cancel()
	_, err = pool.ExecContext(ctx, `DELETE FROM GX_SessionHandoff WHERE UserUID IN
		(SELECT UserUID FROM GX_SessionHandoff WHERE ExpiresAt<=unixepoch() ORDER BY ExpiresAt LIMIT 1024)`)
	return err
}
