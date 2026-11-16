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
	pool.Close()
}
