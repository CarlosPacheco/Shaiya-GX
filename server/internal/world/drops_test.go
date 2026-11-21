package world

import (
	"math"
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/collision"
	"github.com/Spelunkern/Shaiya-GX/internal/mapgeometry"
	"github.com/Spelunkern/Shaiya-GX/server/internal/db"
)

func TestMobDropStaysOnKillFloor(t *testing.T) {
	g := &mapgeometry.Geometry{}
	g.Reindex()
	g.Add(collision.Triangle{{4, 16, 4}, {4, 16, 6}, {6, 17, 4}})
	g.Add(collision.Triangle{{6, 17, 6}, {6, 17, 4}, {4, 16, 6}})
	g.Add(collision.Triangle{{0, 2, 0}, {0, 2, 20}, {20, 2, 0}})
	g.Add(collision.Triangle{{0, 20, 0}, {0, 20, 20}, {20, 20, 0}})

	origin := mapgeometry.Position{5, 16.5, 5}
	candidate := mapgeometry.Position{5.4, 16.5, 5.3}
	if got := floorMobDrop(g, origin, candidate); got[0] != candidate[0] || got[2] != candidate[2] || math.Abs(float64(got[1]-16.7)) > .001 {
		t.Fatalf("expected loot on the sloped middle floor, got %v", got)
	}
	ledge := mapgeometry.Position{5.875, 16.9375, 5}
	if got := floorMobDrop(g, ledge, mapgeometry.Position{6.25, ledge[1], 5}); got != ledge {
		t.Fatalf("loot scattered over a ledge: %v", got)
	}
	for _, tri := range collision.BlockerBox(collision.Vec{5.25, 16, 4}, collision.Vec{5.3, 18, 6}, 0) {
		g.Add(tri)
	}
	if got := floorMobDrop(g, origin, candidate); got != origin {
		t.Fatalf("loot scattered through a wall: %v", got)
	}
}

func TestMobDropQuestAndGroupOverflowKeepKillHeight(t *testing.T) {
	s, _ := forgeRuleSession(t)
	mapID := s.char.MapID
	t.Cleanup(func() { clearMapItems(mapID) })
	defs := db.MustDefs()
	quest := &db.ItemDef{Type: 27, TypeID: 1, Quality: 100}
	defs.Items[db.ItemKey{Type: quest.Type, TypeID: quest.TypeID}] = quest
	oldLinks := questItemLinks
	t.Cleanup(func() { questItemLinks = oldLinks })
	questItemLinks = map[[2]uint8]questItemLink{{quest.Type, quest.TypeID}: {Start: []uint16{1}}}
	defs.MobDrops[1] = []db.MobDrop{{ItemOrder: 1, Grade: 1, DropRate: 100}}
	buildDropCatalog()

	mob := &MapMob{ID: 1, MobID: 1, PosY: -12.5, def: &db.MobDef{QuestItemID: 1}}
	rewardKill(nil, s.char.ID, mapID, mob, s.log)
	ground := itemsOnMap(mapID)
	if len(ground) != 2 {
		t.Fatalf("expected loot and quest item, got %v", ground)
	}
	for _, drop := range ground {
		if drop.PosY != mob.PosY {
			t.Fatalf("item %d/%d lost kill height: %g", drop.Item.Type, drop.Item.TypeID, drop.PosY)
		}
	}
	clearMapItems(mapID)

	p := &ConnectedPlayer{CharacterID: s.char.ID, party: &party{}}
	if drops := rollMobDrops(mapID, mob, s.char.ID, s.char.ID, []*ConnectedPlayer{p}, s.log); len(drops) != 0 {
		t.Fatalf("group item was not queued: %v", drops)
	}
	for _, slot := range freeInventorySlots(s.items) {
		s.items = append(s.items, db.Item{Bag: slot[0], Slot: slot[1], Type: 16, TypeID: 1, Count: 1})
	}
	s.applyLoot(p)
	ground = itemsOnMap(mapID)
	if len(ground) != 1 || ground[0].PosY != mob.PosY {
		t.Fatalf("group overflow did not fall on the kill floor: %v", ground)
	}
}
