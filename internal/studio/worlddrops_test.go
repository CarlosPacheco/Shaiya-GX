package studio

import (
	"path/filepath"
	"slices"
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/worlddrops"
)

func TestSaveWorldDropsPreservesExactChanceAndBackup(t *testing.T) {
	ctx := t.Context()
	root := t.TempDir()
	path := filepath.Join(root, "local.sqlite")
	db, err := openDB(ctx, path, "rwc")
	must(t, err)
	_, err = db.ExecContext(ctx, worlddrops.MigrationSQL)
	must(t, err)
	must(t, db.Close())

	catalog, err := OpenWorldDrops(ctx, path)
	must(t, err)
	if len(catalog.Rules) != 12 || catalog.Rules[0].DropChance != 10000 {
		t.Fatal("unexpected migrated rules")
	}
	draft := slices.Clone(catalog.Rules)
	draft[0].DropChance, err = worlddrops.ParseChance("0.00000001")
	must(t, err)
	draft = slices.Delete(draft, 1, 2)
	draft = append(draft, worlddrops.Rule{Index: 20, Description: "Example", Grade: 34, MinLevel: 100, MaxLevel: 100, DropChance: 0})
	backup, err := SaveWorldDrops(ctx, root, catalog, draft)
	must(t, err)
	before, err := OpenWorldDrops(ctx, filepath.Join(backup, "rapture.sqlite"))
	must(t, err)
	after, err := OpenWorldDrops(ctx, path)
	must(t, err)
	if !slices.Equal(before.Rules, catalog.Rules) || !slices.Equal(after.Rules, draft) ||
		worlddrops.FormatChance(after.Rules[0].DropChance) != "0.00000001" {
		t.Fatal("backup or saved rules differ from their expected values")
	}
	if _, err := SaveWorldDrops(ctx, root, catalog, draft); err == nil {
		t.Fatal("accepted a stale draft")
	}
}
