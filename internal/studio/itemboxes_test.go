package studio

import (
	"path/filepath"
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/itemboxes"
	"github.com/Spelunkern/Shaiya-GX/internal/worlddrops"
)

func TestItemBoxesSaveAndBackUpBothTables(t *testing.T) {
	ctx := t.Context()
	root := t.TempDir()
	path := filepath.Join(root, "boxes.sqlite")
	db, err := openDB(ctx, path, "rwc")
	must(t, err)
	_, err = db.ExecContext(ctx, `CREATE TABLE PS_GameDefs_Items (ItemID INTEGER,Type INTEGER,TypeID INTEGER,Grade INTEGER,Special INTEGER,ReqVg INTEGER);
		INSERT INTO PS_GameDefs_Items VALUES (440204,44,204,99,42,9),(30001,30,1,51,0,0),(30002,30,2,52,0,0);`+itemboxes.MigrationSQL)
	must(t, err)
	must(t, db.Close())
	old, err := OpenItemBoxes(ctx, path)
	must(t, err)
	if len(old.Rules) != 1 || old.Rules[0].Rewards[0].Chance != 7 {
		t.Fatal("original weights were not migrated")
	}
	draft := itemboxes.Clone(old.Rules)
	draft[0].Description = "Edited"
	draft = append(draft, itemboxes.Box{Index: 10, BoxType: itemboxes.Independent, ItemType: itemboxes.Direct,
		Rewards: []itemboxes.Reward{{Item: 30001, Chance: worlddrops.RollRange}, {Item: 30002, Chance: 1}}})
	backup, err := SaveItemBoxes(ctx, root, old, draft)
	must(t, err)
	before, err := OpenItemBoxes(ctx, filepath.Join(backup, "rapture.sqlite"))
	must(t, err)
	after, err := OpenItemBoxes(ctx, path)
	must(t, err)
	if !itemboxes.Equal(before.Rules, old.Rules) || !itemboxes.Equal(after.Rules, draft) {
		t.Fatal("box or reward data differs")
	}
	if _, err := SaveItemBoxes(ctx, root, old, draft); err == nil {
		t.Fatal("stale draft saved")
	}
}
