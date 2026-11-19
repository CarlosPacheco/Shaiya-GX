package studio

import (
	"path/filepath"
	"slices"
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/blessingrules"
)

func TestSaveBlessingMirrorsBonusesAndBacksUpBothTables(t *testing.T) {
	ctx := t.Context()
	root := t.TempDir()
	path := filepath.Join(root, "local.sqlite")
	db, err := openDB(ctx, path, "rwc")
	must(t, err)
	_, err = db.ExecContext(ctx, blessingrules.MigrationSQL)
	must(t, err)
	must(t, db.Close())

	old, err := OpenBlessing(ctx, path)
	must(t, err)
	draft := blessingrules.Catalog{slices.Clone(old.Rules[0]), slices.Clone(old.Rules[1])}
	draft[0][0].Bonuses[0] = 42
	if _, err := SaveBlessing(ctx, root, old, draft); err == nil {
		t.Fatal("accepted unequal bonuses")
	}
	draft[1][0].Bonuses[0] = 42
	draft[0][0].RequiredKills--
	backup, err := SaveBlessing(ctx, root, old, draft)
	must(t, err)
	before, err := OpenBlessing(ctx, filepath.Join(backup, "rapture.sqlite"))
	must(t, err)
	after, err := OpenBlessing(ctx, path)
	must(t, err)
	if !EqualBlessing(before.Rules, old.Rules) || !EqualBlessing(after.Rules, draft) {
		t.Fatal("backup or saved rules differ from their expected value")
	}
	if after.Rules.Effective(1, 2)[0] != 42 {
		t.Fatal("effective preview lost inherited bonus")
	}
	if _, err := SaveBlessing(ctx, root, old, draft); err == nil {
		t.Fatal("accepted stale updates")
	}
}
