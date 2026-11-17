package db

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"fmt"
	"time"
)

type migration struct {
	Version   int
	Name, SQL string
}

var migrations = []migration{
	{1, "atomic_session_handoff", `CREATE TABLE IF NOT EXISTS GX_SessionHandoff(UserUID INTEGER PRIMARY KEY,Identity BLOB NOT NULL,AESMaterial BLOB NOT NULL,SessionID INTEGER NOT NULL,ExpiresAt INTEGER NOT NULL)`},
	{2, "indexed_account_lookup", `CREATE INDEX IF NOT EXISTS GX_Users_UserID ON PS_UserData_Users_Master(UserID)`},
	{3, "credential_revocation", `CREATE TABLE GX_AccountSecurity(UserUID INTEGER PRIMARY KEY,Version INTEGER NOT NULL DEFAULT 1); ALTER TABLE GX_SessionHandoff ADD COLUMN CredentialVersion INTEGER NOT NULL DEFAULT 1`},
	{4, "warehouse_lookup", `CREATE INDEX GX_Warehouse_User ON PS_GameData_UserStoredItems(ServerID,UserUID,ItemUID)`},
	{5, "active_account_ownership", `CREATE TABLE GX_ActiveSessions(UserUID INTEGER PRIMARY KEY,SessionID INTEGER NOT NULL)`},
	{6, "character_teleport_points", `CREATE TABLE GX_TeleportPoints(
		CharID INTEGER NOT NULL REFERENCES PS_GameData_Chars(CharID) ON DELETE CASCADE,
		Slot INTEGER NOT NULL CHECK(Slot BETWEEN 0 AND 3),
		Map INTEGER NOT NULL CHECK(Map BETWEEN 0 AND 65535),
		X REAL NOT NULL, Y REAL NOT NULL, Z REAL NOT NULL,
		PRIMARY KEY(CharID, Slot))`},
}

// Base user_version=1 remains compatible with the distributed development seed.
// GX's additive migrations have their own transactional, checksummed ledger.
func migrate(ctx context.Context, pool *sql.DB, steps []migration) error {
	tx, err := pool.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err = tx.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS GX_SchemaMigrations(Version INTEGER PRIMARY KEY,Name TEXT NOT NULL,Checksum TEXT NOT NULL,AppliedAt TEXT NOT NULL)`); err != nil {
		return err
	}
	var highest int
	if err = tx.QueryRowContext(ctx, `SELECT COALESCE(MAX(Version),0) FROM GX_SchemaMigrations`).Scan(&highest); err != nil {
		return err
	}
	if highest > len(steps) {
		return fmt.Errorf("runtime migration %d is newer than this server", highest)
	}
	for i, step := range steps {
		if step.Version != i+1 {
			return fmt.Errorf("non-contiguous migration version")
		}
		checksum := fmt.Sprintf("%x", sha256.Sum256([]byte(step.SQL)))
		var oldName, oldChecksum string
		err = tx.QueryRowContext(ctx, `SELECT Name,Checksum FROM GX_SchemaMigrations WHERE Version=?`, step.Version).Scan(&oldName, &oldChecksum)
		if err == nil {
			if oldName != step.Name || oldChecksum != checksum {
				return fmt.Errorf("migration %d checksum mismatch", step.Version)
			}
			continue
		}
		if err != sql.ErrNoRows {
			return err
		}
		if step.Version <= highest {
			return fmt.Errorf("migration history contains gaps")
		}
		if _, err = tx.ExecContext(ctx, step.SQL); err != nil {
			return fmt.Errorf("migration %d: %w", step.Version, err)
		}
		if _, err = tx.ExecContext(ctx, `INSERT INTO GX_SchemaMigrations VALUES(?,?,?,?)`, step.Version, step.Name, checksum, time.Now().UTC().Format(time.RFC3339)); err != nil {
			return err
		}
	}
	return tx.Commit()
}
