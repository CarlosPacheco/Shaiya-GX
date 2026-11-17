package world

import (
	"github.com/Spelunkern/Shaiya-GX/server/internal/db"
	"testing"
)

func TestEquippedItemExpAbsorption(t *testing.T) {
	old := db.MustDefs()
	t.Cleanup(func() { db.SetDefsForTest(old) })
	db.SetDefsForTest(&db.Defs{Items: map[db.ItemKey]*db.ItemDef{
		{Type: 17, TypeID: 1}: {Type: 17, TypeID: 1, ExpAbsorb: 30, Server: 1, ReqWis: 1},
		{Type: 22, TypeID: 1}: {Type: 22, TypeID: 1, ExpAbsorb: 255},
	}})
	items := []db.Item{
		{Bag: 0, Slot: 1, Type: 17, TypeID: 1, CraftName: "00000000000000000053"},
		{Bag: 0, Slot: 8, Type: 22, TypeID: 1},
		{Bag: 1, Slot: 0, Type: 22, TypeID: 1},
	}
	// Items.Exp stacks with other equipped items and the armor +3 enchant (15).
	equip := GetEquipmentStats(items)
	if got := StatsFor(&db.Character{Level: 1}, equip).Absorption; got != 300 {
		t.Fatalf("absorption=%d, want 300", got)
	}
	// Removing the armor removes both its native and enchanted absorption.
	items[0].Bag = 1
	if got := GetEquipmentStats(items).Absorption; got != 255 {
		t.Fatalf("after unequip=%d, want 255", got)
	}
	items[1].Bag = 1
	if got := GetEquipmentStats(items).Absorption; got != 0 {
		t.Fatalf("carried items grant absorption=%d", got)
	}
}
