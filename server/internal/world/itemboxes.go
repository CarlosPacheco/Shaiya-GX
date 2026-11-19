package world

import (
	"context"
	"fmt"
	"math/rand/v2"
	"slices"
	"sync/atomic"

	"github.com/Spelunkern/Shaiya-GX/internal/itemboxes"
	"github.com/Spelunkern/Shaiya-GX/internal/worlddrops"
	"github.com/Spelunkern/Shaiya-GX/server/internal/db"
	"github.com/Spelunkern/Shaiya-GX/server/internal/logx"
)

const specialItemBox = 42

type boxReward struct {
	items  []*db.ItemDef
	chance uint64
}

type itemBox struct {
	boxType uint8
	total   uint64
	rewards []boxReward
}

var itemBoxCatalog atomic.Pointer[map[int16]itemBox]

func loadItemBoxes(log *logx.Logger) error {
	pool, err := db.Pool(db.GameDefs)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), db.Timeout)
	defer cancel()
	boxes, err := itemboxes.Read(ctx, pool)
	if err != nil {
		return err
	}
	catalog := compileItemBoxes(boxes, db.MustDefs(), log)
	itemBoxCatalog.Store(&catalog)
	log.Printf("GX Server World loaded item boxes=%d", len(catalog))
	return nil
}

func compileItemBoxes(boxes []itemboxes.Box, defs *db.Defs, log logPrinter) map[int16]itemBox {
	byID := make(map[int32][]*db.ItemDef)
	for _, def := range defs.Items {
		if def.Type != 0 && def.TypeID != 0 {
			byID[def.ItemID] = append(byID[def.ItemID], def)
		}
	}
	catalog := make(map[int16]itemBox, len(boxes))
	for _, box := range boxes {
		runtime := itemBox{boxType: box.BoxType, total: box.Total()}
		valid := true
		for _, reward := range box.Rewards {
			if reward.Chance == 0 {
				continue
			}
			var items []*db.ItemDef
			switch box.ItemType {
			case itemboxes.Grade:
				for _, def := range defs.ItemsByGrade[uint16(reward.Item)] {
					if !isQuestItemType(def.Type) {
						items = append(items, def)
					}
				}
			case itemboxes.Direct:
				if exact := byID[reward.Item]; len(exact) == 1 {
					items = exact
				}
			}
			if len(items) == 0 {
				log.Printf("GX Server World disabled item box index=%d: unresolved reward %d", box.Index, reward.Item)
				valid = false
				break
			}
			runtime.rewards = append(runtime.rewards, boxReward{items: items, chance: reward.Chance})
		}
		if valid {
			catalog[box.Index] = runtime
		}
	}
	return catalog
}

func (b itemBox) roll(draw func(uint64) uint64, award func(*db.ItemDef)) {
	if b.boxType == itemboxes.Normal {
		pick := draw(b.total)
		for _, r := range b.rewards {
			if pick < r.chance {
				award(r.items[draw(uint64(len(r.items)))])
				return
			}
			pick -= r.chance
		}
		return
	}
	for _, r := range b.rewards {
		if draw(worlddrops.RollRange) < r.chance {
			award(r.items[draw(uint64(len(r.items)))])
		}
	}
}

func boxItem(def *db.ItemDef) db.Item {
	return db.Item{Type: def.Type, TypeID: def.TypeID, Quality: def.Quality, Count: 1}
}

func (b itemBox) canOpen(items []db.Item) bool {
	free := freeInventorySlots(items)
	if b.boxType == itemboxes.Independent {
		if len(free) >= len(b.rewards) {
			return true
		}
		// Reserve capacity before drawing. Deterministic rewards can share
		// stacks. A grade with multiple candidates reserves one free slot.
		sim := slices.Clone(items)
		for _, reward := range b.rewards {
			if len(reward.items) == 1 {
				tpl := boxItem(reward.items[0])
				if i := findStackable(sim, &tpl, maxStackOf(tpl.Type, tpl.TypeID)); i >= 0 {
					sim[i].Count++
					continue
				}
				if len(free) == 0 {
					return false
				}
				tpl.Bag, tpl.Slot = free[0][0], free[0][1]
				sim = append(sim, tpl)
			} else if len(free) == 0 {
				return false
			}
			free = free[1:]
		}
		return true
	}
	if len(free) > 0 {
		return true
	}
	for _, reward := range b.rewards {
		for _, def := range reward.items {
			item := boxItem(def)
			if findStackable(items, &item, maxStackOf(item.Type, item.TypeID)) < 0 {
				return false
			}
		}
	}
	return true
}

func (s *Session) openItemBox(slot int, index int16) bool {
	catalog := itemBoxCatalog.Load()
	if catalog == nil || s.items[slot].Bag == 0 || s.dead || s.char.CurrentHP == 0 {
		return false
	}
	box, ok := (*catalog)[index]
	if !ok {
		return false
	}

	return s.inventoryOperation(func() {
		consumed, err := s.consumeOne(slot)
		if err != nil {
			s.operation.err = err
			return
		}
		if !box.canOpen(s.items) {
			s.operation.err = fmt.Errorf("insufficient inventory space for item box %d", index)
			return
		}

		s.send(buildUseItemResult(s.char.ID, &consumed), "item box consumed")
		beforeRewards := slices.Clone(s.items)
		box.roll(rand.Uint64N, func(def *db.ItemDef) {
			if s.operation.err != nil {
				return
			}
			tpl := boxItem(def)
			_, left := planGiveItem(&s.items, &tpl, 1, maxStackOf(tpl.Type, tpl.TypeID))
			if left != 0 {
				s.operation.err = fmt.Errorf("item box %d reward did not fit", index)
				return
			}
		})
		for _, item := range s.items {
			i := findItem(beforeRewards, item.Bag, item.Slot)
			if i < 0 || beforeRewards[i] != item {
				s.send(buildAddItem(&item), "item box reward")
			}
		}
	})
}
