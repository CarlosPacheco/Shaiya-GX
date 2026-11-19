package studio

import (
	"path/filepath"
	"reflect"
	"slices"
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/maprules"
)

func TestSaveMapRulesBacksUpPropertiesAndRaceStarts(t *testing.T) {
	ctx := t.Context()
	root := t.TempDir()
	path := filepath.Join(root, "maps.sqlite")
	db, err := openDB(ctx, path, "rwc")
	must(t, err)
	_, err = db.ExecContext(ctx, maprules.MigrationSQL)
	must(t, err)
	must(t, db.Close())
	before, err := OpenMapRules(ctx, path)
	must(t, err)
	draft := maprules.Catalog{Maps: slices.Clone(before.Rules.Maps), Starts: slices.Clone(before.Rules.Starts)}
	draft.Maps[18] = slices.Clone(draft.Maps[18])
	draft.Maps[18][10] = int64(50)
	draft.Starts[0] = slices.Clone(draft.Starts[0])
	draft.Starts[0][2] = float64(578)
	backup, err := SaveMapRules(ctx, root, before, draft)
	must(t, err)
	old, err := OpenMapRules(ctx, filepath.Join(backup, "rapture.sqlite"))
	must(t, err)
	after, err := OpenMapRules(ctx, path)
	must(t, err)
	if !reflect.DeepEqual(before.Rules, old.Rules) || !reflect.DeepEqual(after.Rules, draft) {
		t.Fatal("map save or backup differs")
	}
	draft.Starts[0][1] = int64(18)
	if _, err := SaveMapRules(ctx, root, after, draft); err == nil {
		t.Fatal("a restricted map cannot be a birth fallback")
	}
}
