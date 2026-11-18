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
	{7, "skill_catalog_cleanup", `-- Dead skill levels (placeholders such as "...ED" / "Not Available.") of skills that have
		-- working levels. Fully dead skills and every working level stay.
		DELETE FROM PS_GameDefs_Skills WHERE SkillLevel BETWEEN 1 AND 9 AND (SkillID,SkillLevel) IN (VALUES
		(174,3),(201,2),(201,3),(202,2),(202,3),(203,2),(203,3),(204,2),(204,3),(205,2),
		(205,3),(206,2),(206,3),(210,2),(210,3),(211,2),(211,3),(212,2),(212,3),(213,2),
		(213,3),(214,2),(214,3),(215,2),(215,3),(222,2),(222,3),(223,2),(223,3),(224,2),
		(224,3),(225,2),(225,3),(226,2),(226,3),(227,2),(227,3),(228,2),(228,3),(229,2),
		(229,3),(230,2),(230,3),(231,2),(231,3),(232,2),(232,3),(233,2),(233,3),(234,2),
		(234,3),(235,2),(235,3),(236,2),(236,3),(237,2),(237,3),(238,2),(238,3),(239,2),
		(239,3),(240,2),(240,3),(241,2),(241,3),(242,2),(242,3),(243,2),(243,3),(244,2),
		(244,3),(245,2),(245,3),(246,2),(246,3),(247,2),(247,3),(248,2),(248,3),(249,2),
		(249,3),(250,3),(251,2),(251,3),(252,2),(252,3),(253,2),(253,3),(254,2),(254,3),
		(255,2),(255,3),(261,2),(261,3),(262,2),(262,3),(266,3),(267,2),(267,3),(268,3),
		(269,3),(270,3),(271,2),(271,3),(272,2),(272,3),(273,3),(302,3),(309,3),(310,3),
		(320,2),(320,3),(321,3),(322,3),(323,3),(324,3));
		-- The chain mechanic is not used: its previous-skill relation goes (Shadow Attack and
		-- Shadow Sleep keep the Stealth requirement in their own previous-skill field).
		UPDATE PS_GameDefs_Skills SET PrevSkillID=0 WHERE SkillLevel<100 AND SkillID IN (61,64,79,94,104,135,145,150,161,302,311,312,313);
		-- Aggravation's time damage is not part of the skill.
		UPDATE PS_GameDefs_Skills SET TimeDamageHP=0,TimeDamageSP=0,TimeDamageMP=0 WHERE SkillID=150 AND SkillLevel<100;
		-- Detail 25 was a common attack with extra HP/MP/SP damage; AddDamage is read from its own
		-- fields, so the standard attack detail 21 behaves identically.
		UPDATE PS_GameDefs_Skills SET TypeDetail=21 WHERE TypeDetail=25 AND TypeEffect=1 AND SkillLevel<100`},
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
