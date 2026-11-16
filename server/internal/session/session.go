// Package session hands an authenticated login over to the world server.
// Login writes an atomic SQLite ticket; the client then connects to
// world and presents the same identity token in its 0xA301 handshake.
package session

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"database/sql"
	"encoding/binary"
	"errors"
	"github.com/Spelunkern/Shaiya-GX/server/internal/db"
	"time"
)

// MaxAge bounds how long a handoff stays valid after login.
const MaxAge = 10 * time.Minute

type Handoff struct {
	Identity    [16]byte
	AESMaterial [32]byte // login AES key (16) + IV (16)
	UserUID     uint32
	SessionID   int64
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
	_, err = pool.ExecContext(ctx, `INSERT INTO GX_SessionHandoff
 (UserUID,Identity,AESMaterial,SessionID,ExpiresAt) VALUES(?,?,?,?,?)
 ON CONFLICT(UserUID) DO UPDATE SET Identity=excluded.Identity,
 AESMaterial=excluded.AESMaterial,SessionID=excluded.SessionID,ExpiresAt=excluded.ExpiresAt`,
		h.UserUID, h.Identity[:], h.AESMaterial[:], h.SessionID, time.Now().Add(MaxAge).Unix())
	return err
}

// Consume validates and deletes a ticket in one write transaction, including
// concurrent callers in different processes. Invalid identities cannot burn it.
func Consume(userUID uint32, identity [16]byte) (Handoff, error) {
	pool, err := db.Pool(db.UserData)
	if err != nil {
		return Handoff{}, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), db.Timeout)
	defer cancel()
	tx, err := pool.BeginTx(ctx, nil)
	if err != nil {
		return Handoff{}, err
	}
	defer tx.Rollback()
	var stored, material []byte
	var expires int64
	h := Handoff{UserUID: userUID}
	err = tx.QueryRowContext(ctx, `SELECT Identity,AESMaterial,SessionID,ExpiresAt
 FROM GX_SessionHandoff WHERE UserUID=?`, userUID).Scan(&stored, &material, &h.SessionID, &expires)
	if errors.Is(err, sql.ErrNoRows) {
		return Handoff{}, ErrInvalid
	}
	if err != nil {
		return Handoff{}, err
	}
	if len(stored) != 16 || len(material) != 32 || expires <= time.Now().Unix() || subtle.ConstantTimeCompare(stored, identity[:]) != 1 {
		return Handoff{}, ErrInvalid
	}
	copy(h.Identity[:], stored)
	copy(h.AESMaterial[:], material)
	if _, err = tx.ExecContext(ctx, `DELETE FROM GX_SessionHandoff WHERE UserUID=?`, userUID); err != nil {
		return Handoff{}, err
	}
	if err = tx.Commit(); err != nil {
		return Handoff{}, err
	}
	return h, nil
}
