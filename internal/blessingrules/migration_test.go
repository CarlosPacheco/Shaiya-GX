package blessingrules

import (
	"database/sql"
	"path/filepath"
	"strings"
	"testing"

	_ "modernc.org/sqlite"
)

func TestMergeRejectsDifferentThresholdsAndPreservesMatchingEdits(t *testing.T) {
	ctx := t.Context()
	db, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "game.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	exec := func(query string) {
		t.Helper()
		if _, err := db.ExecContext(ctx, query); err != nil {
			t.Fatal(err)
		}
	}
	exec(MigrationSQL)
	exec("UPDATE PS_GameDefs_BlessingFury SET RequiredKills=175, MaxHPPct=23 WHERE Tier=1")
	merge := func() error {
		tx, err := db.BeginTx(ctx, nil)
		if err != nil {
			return err
		}
		defer tx.Rollback()
		if _, err := tx.ExecContext(ctx, MergeSQL); err != nil {
			return err
		}
		return tx.Commit()
	}
	if err := merge(); err == nil || !strings.Contains(err.Error(), "BlessingLightAndFuryMustMatch") {
		t.Fatalf("expected mismatch error, got %v", err)
	}
	exec("UPDATE PS_GameDefs_BlessingLight SET RequiredKills=175, MaxHPPct=23 WHERE Tier=1")
	if err := merge(); err != nil {
		t.Fatal(err)
	}
	rules, err := Read(ctx, db)
	if err != nil || len(rules) != 30 || rules[0].RequiredKills != 175 || rules[0].Bonuses[0] != 23 {
		t.Fatalf("matching edits were not preserved: %v", err)
	}
	var oldTables int
	if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM sqlite_schema WHERE name IN ('PS_GameDefs_BlessingLight','PS_GameDefs_BlessingFury')").Scan(&oldTables); err != nil || oldTables != 0 {
		t.Fatalf("unexpected old tables: %d, %v", oldTables, err)
	}
}
