package session

import (
	"github.com/Spelunkern/Shaiya-GX/server/internal/db"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestHandoffExactlyOnce(t *testing.T) {
	seed, err := filepath.Abs("../../data/rapture.sqlite")
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("RAPTURE_DB_PATH", filepath.Join(t.TempDir(), "runtime.sqlite"))
	t.Setenv("RAPTURE_DB_SEED", seed)
	h := Handoff{UserUID: 990999, Identity: NewIdentity(), SessionID: NewSessionID()}
	if err := Save(h); err != nil {
		t.Fatal(err)
	}
	if _, err := Consume(h.UserUID, NewIdentity()); err != ErrInvalid {
		t.Fatal("invalid token accepted", err)
	}
	var wins atomic.Int32
	var wg sync.WaitGroup
	for range 24 {
		wg.Go(func() {
			if _, err := Consume(h.UserUID, h.Identity); err == nil {
				wins.Add(1)
			} else if err != ErrInvalid {
				t.Error(err)
			}
		})
	}
	wg.Wait()
	if wins.Load() != 1 {
		t.Fatal("consumed", wins.Load(), "times")
	}
	if err := Save(h); err != nil {
		t.Fatal(err)
	}
	pool, err := db.Pool(db.UserData)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = pool.Exec(`UPDATE GX_SessionHandoff SET ExpiresAt=?`, time.Now().Add(-time.Second).Unix()); err != nil {
		t.Fatal(err)
	}
	if _, err = Consume(h.UserUID, h.Identity); err != ErrInvalid {
		t.Fatal("expired token accepted", err)
	}
	db.Close()
}

func TestCredentialChangeRevokesPendingAndStaleTickets(t *testing.T) {
	db.Close()
	seed, err := filepath.Abs("../../data/rapture.sqlite")
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("RAPTURE_DB_PATH", filepath.Join(t.TempDir(), "runtime.sqlite"))
	t.Setenv("RAPTURE_DB_SEED", seed)
	t.Cleanup(func() { db.Close() })
	uid, err := db.CreateAccount("revoke_account", "old passphrase", false)
	if err != nil {
		t.Fatal(err)
	}
	login, err := db.TryGameLogin("revoke_account", "old passphrase", 1, "127.0.0.1")
	if err != nil {
		t.Fatal(err)
	}
	h := Handoff{UserUID: uint32(uid), Identity: NewIdentity(), SessionID: 1, CredentialVersion: login.CredentialVersion}
	if err := Save(h); err != nil {
		t.Fatal(err)
	}
	if err := db.SetAccountPassword("revoke_account", "new passphrase"); err != nil {
		t.Fatal(err)
	}
	if _, err := Consume(uint32(uid), h.Identity); err != ErrInvalid {
		t.Fatal("revoked ticket accepted", err)
	}
	if err := Save(h); err != ErrInvalid {
		t.Fatal("stale authentication recreated a ticket", err)
	}
	login, err = db.TryGameLogin("revoke_account", "new passphrase", 2, "127.0.0.1")
	if err != nil {
		t.Fatal(err)
	}
	h.SessionID = 2
	h.Identity = NewIdentity()
	h.CredentialVersion = login.CredentialVersion
	if err := Save(h); err != nil {
		t.Fatal(err)
	}
	if _, err := Consume(uint32(uid), h.Identity); err != nil {
		t.Fatal(err)
	}
}
