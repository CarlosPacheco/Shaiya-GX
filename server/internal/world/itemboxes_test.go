package world

import (
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/itemboxes"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/worlddrops"
	"github.com/Spelunkern/Shaiya-GX/server/internal/db"
)

func TestItemBoxRollModes(t *testing.T) {
	first, second := &db.ItemDef{ItemID: 1}, &db.ItemDef{ItemID: 2}
	box := itemBox{boxType: itemboxes.Normal, total: 1500, rewards: []boxReward{{[]*db.ItemDef{first}, 900}, {[]*db.ItemDef{second}, 600}}}
	for _, tc := range []struct {
		draw uint64
		want *db.ItemDef
	}{{899, first}, {900, second}} {
		var rewards []*db.ItemDef
		box.roll(func(limit uint64) uint64 {
			if limit == 1 {
				return 0
			}
			if limit != 1500 {
				t.Fatal(limit)
			}
			return tc.draw
		}, func(def *db.ItemDef) { rewards = append(rewards, def) })
		if len(rewards) != 1 || rewards[0] != tc.want {
			t.Fatalf("wrong normalized outcome at %d", tc.draw)
		}
	}
	box.boxType = itemboxes.Independent
	box.rewards[0].chance, box.rewards[1].chance = 100, worlddrops.RollRange
	var count, draws int
	box.roll(func(limit uint64) uint64 {
		if limit == 1 {
			return 0
		}
		if limit != worlddrops.RollRange {
			t.Fatal(limit)
		}
		draws++
		return 100
	}, func(def *db.ItemDef) {
		count++
		if def != second {
			t.Fatal("failed probability awarded a reward")
		}
	})
	if count != 1 || draws != 2 {
		t.Fatalf("draws=%d awards=%d", draws, count)
	}
}

func TestUseItemBoxConsumesAndGrantsInOneInventoryOperation(t *testing.T) {
	s, add := forgeRuleSession(t)
	defs := db.MustDefs()
	bag := defs.Item(25, 43)
	bag.Special, bag.ReqVg = specialItemBox, 9
	first, second := defs.Item(30, 1), defs.Item(30, 2)
	first.ItemID, second.ItemID = 30001, 30002
	item := add(db.Item{Type: bag.Type, TypeID: bag.TypeID, Bag: 1, Slot: 1, Count: 1})
	req := proto.UseItemPacket(item.Bag, item.Slot)
	old := itemBoxCatalog.Load()
	t.Cleanup(func() { itemBoxCatalog.Store(old) })
	empty := map[int16]itemBox{}
	itemBoxCatalog.Store(&empty)
	s.handleUseItem(req)
	if len(s.items) != 1 || s.items[0].UID != item.UID {
		t.Fatal("missing configuration consumed box")
	}
	boxes := []itemboxes.Box{{Index: 9, BoxType: itemboxes.Independent, ItemType: itemboxes.Direct,
		Rewards: []itemboxes.Reward{{Item: 30001, Chance: worlddrops.RollRange}, {Item: 30002, Chance: worlddrops.RollRange}}}}
	catalog := compileItemBoxes(boxes, defs, nopLog{})
	itemBoxCatalog.Store(&catalog)
	s.handleUseItem(req)
	items, err := db.CharacterItems(s.char.ID)
	if err != nil || len(items) != 2 || len(s.items) != 2 {
		t.Fatalf("expected both awards: %v %v", items, err)
	}
	for _, award := range items {
		if award.Type != 30 || award.UID == item.UID {
			t.Fatalf("unexpected award %+v", award)
		}
	}
}

func TestItemBoxKeepsBagWhenRewardsCannotFit(t *testing.T) {
	s, add := forgeRuleSession(t)
	defs := db.MustDefs()
	bag := defs.Item(25, 43)
	bag.Special, bag.ReqVg = specialItemBox, 9
	target := defs.Item(1, 1)
	target.ItemID = 1001
	item := add(db.Item{Type: bag.Type, TypeID: bag.TypeID, Bag: 1, Slot: 0, Count: 2})
	for _, slot := range freeInventorySlots(s.items) {
		add(db.Item{Type: 30, TypeID: 1, Bag: slot[0], Slot: slot[1], Count: 1})
	}
	size := len(s.items)
	old := itemBoxCatalog.Load()
	t.Cleanup(func() { itemBoxCatalog.Store(old) })
	boxes := []itemboxes.Box{{Index: 9, BoxType: itemboxes.Normal, ItemType: itemboxes.Direct,
		Rewards: []itemboxes.Reward{{Item: 1001, Chance: 1000}}}}
	catalog := compileItemBoxes(boxes, defs, nopLog{})
	itemBoxCatalog.Store(&catalog)
	s.handleUseItem(proto.UseItemPacket(item.Bag, item.Slot))
	items, err := db.CharacterItems(s.char.ID)
	if err != nil || len(items) != size || len(s.items) != size {
		t.Fatalf("inventory changed: %d %v", len(items), err)
	}
	for _, inventory := range [][]db.Item{items, s.items} {
		i := findItem(inventory, item.Bag, item.Slot)
		if i < 0 || inventory[i].UID != item.UID || inventory[i].Count != 2 {
			t.Fatal("box stack changed without space")
		}
	}
}
