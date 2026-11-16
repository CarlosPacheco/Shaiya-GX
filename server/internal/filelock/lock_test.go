package filelock

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"
)

func TestLockExclusionCancellationAndRelease(t *testing.T) {
	path := filepath.Join(t.TempDir(), "init.lock")
	release, err := Acquire(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()
	if another, err := Acquire(ctx, path); !errors.Is(err, context.DeadlineExceeded) {
		if another != nil {
			another()
		}
		release()
		t.Fatal(err)
	}
	release()
	released, err := Acquire(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}
	released()
}
