package world

import (
	"github.com/Spelunkern/Shaiya-GX/server/internal/db"
	"testing"
)

func TestMobQuestDropsUseNativeQuestTypes(t *testing.T) {
	oldDefs, oldLinks := db.MustDefs(), questItemLinks
	t.Cleanup(func() { db.SetDefsForTest(oldDefs); questItemLinks = oldLinks })
	defs := *oldDefs
	defs.Items = map[db.ItemKey]*db.ItemDef{}
	questItemLinks = map[[2]uint8]questItemLink{}
	db.SetDefsForTest(&defs)
	for _, tt := range []struct {
		index int32
		key   db.ItemKey
	}{{248, db.ItemKey{Type: 27, TypeID: 248}}, {1001, db.ItemKey{Type: 28, TypeID: 1}}, {2001, db.ItemKey{Type: 29, TypeID: 1}}} {
		item := &db.ItemDef{Type: tt.key.Type, TypeID: tt.key.TypeID, Quality: 100}
		defs.Items[tt.key] = item
		m := &MapMob{def: &db.MobDef{QuestItemID: tt.index}}
		if got := questItemDrop(m, nil); got != nil {
			t.Fatal("unlinked quest item dropped")
		}
		questItemLinks[[2]uint8{tt.key.Type, tt.key.TypeID}] = questItemLink{Start: []uint16{1}}
		if got := questItemDrop(m, nil); got != item {
			t.Fatalf("native quest index %d did not drop linked item %+v", tt.index, tt.key)
		}
	}
}
