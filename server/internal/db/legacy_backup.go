package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/Spelunkern/Shaiya-GX/server/internal/config"
	"github.com/Spelunkern/Shaiya-GX/server/internal/filelock"
)

const pruneLegacyVersion = 26

// Keep the pre-prune schema available to the legacy exporters. A separate
// process lock serializes backup and migration across Login and World.
func migrateRuntime(ctx context.Context, pool *sql.DB) error {
	path := config.DatabasePath()
	unlock, err := filelock.Acquire(ctx, path+".schema.lock")
	if err != nil {
		return err
	}
	defer unlock()
	var hasLedger bool
	if err := pool.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM sqlite_schema WHERE type='table' AND name='GX_SchemaMigrations')`).Scan(&hasLedger); err != nil {
		return err
	}
	var version int
	if hasLedger {
		if err := pool.QueryRowContext(ctx, `SELECT COALESCE(MAX(Version),0) FROM GX_SchemaMigrations`).Scan(&version); err != nil {
			return err
		}
	}
	if version < pruneLegacyVersion {
		if err := backupLegacyRuntime(ctx, pool, path); err != nil {
			return fmt.Errorf("backup before legacy SQLite cleanup: %w", err)
		}
	}
	return migrate(ctx, pool, migrations)
}

func backupLegacyRuntime(ctx context.Context, pool *sql.DB, path string) error {
	dir, err := os.MkdirTemp(filepath.Dir(path), ".gx-before-schema-26-")
	if err != nil {
		return err
	}
	verified := false
	defer func() {
		if !verified {
			_ = os.RemoveAll(dir)
		}
	}()
	archive := filepath.Join(dir, filepath.Base(path))
	file, err := os.OpenFile(archive, os.O_CREATE|os.O_EXCL|os.O_RDWR, 0600)
	if err != nil {
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	if _, err := pool.ExecContext(ctx, "VACUUM INTO ?", archive); err != nil {
		return err
	}
	if err := verifyLegacyBackup(ctx, archive); err != nil {
		return err
	}
	for _, target := range []string{archive, dir, filepath.Dir(path)} {
		if runtime.GOOS == "windows" && target != archive {
			continue
		}
		flags := os.O_RDONLY
		if target == archive {
			flags = os.O_RDWR
		}
		handle, err := os.OpenFile(target, flags, 0)
		if err != nil {
			return err
		}
		if err := errors.Join(handle.Sync(), handle.Close()); err != nil {
			return err
		}
	}
	verified = true
	return nil
}

func verifyLegacyBackup(ctx context.Context, archive string) error {
	path, err := filepath.Abs(archive)
	if err != nil {
		return err
	}
	path = filepath.ToSlash(path)
	if runtime.GOOS == "windows" && !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	u := url.URL{Scheme: "file", Path: path, RawQuery: "mode=ro"}
	backup, err := sql.Open("sqlite", u.String())
	if err != nil {
		return err
	}
	defer backup.Close()
	var check string
	if err := backup.QueryRowContext(ctx, "PRAGMA integrity_check").Scan(&check); err != nil {
		return err
	}
	if check != "ok" {
		return fmt.Errorf("backup integrity check: %s", check)
	}
	return nil
}
