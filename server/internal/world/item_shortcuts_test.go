package world

import (
	"fmt"
	"slices"
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/game"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/server/internal/db"
)

func TestConsumedStackClearsAllItsShortcuts(t *testing.T) {
	for _, staged := range []bool{false, true} {
		t.Run(fmt.Sprintf("staged=%t", staged), func(t *testing.T) {
			fixture, add := forgeRuleSession(t)
			item := add(db.Item{Bag: 1, Slot: 4, Type: 25, TypeID: 43, Count: 2})
			db.MustDefs().Item(item.Type, item.TypeID).ConstHP = 10
			s, wire := gmWireSession(t, *fixture.char, false)
			s.items = fixture.items
			keep := db.QuickSlot{QuickSlot: 2, Bag: 100, Number: 1}
			first := db.QuickSlot{Bag: item.Bag, Number: uint16(item.Slot)}
			second := first
			second.QuickBar = 1
			s.quickSlots = []db.QuickSlot{first, keep, second}
			if err := db.SaveQuickSlots(s.char.ID, s.quickSlots); err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = db.SaveQuickSlots(s.char.ID, nil) })
			use := func() {
				t.Helper()
				if !staged {
					s.handleUseItem(proto.UseItemPacket(item.Bag, item.Slot))
				} else if !s.inventoryOperation(func() {
					_, s.operation.err = s.consumeOne(0)
				}) {
					t.Fatal("staged consumption failed")
				}
			}
			use()
			if len(s.quickSlots) != 3 || len(s.items) != 1 || s.items[0].Count != 1 {
				t.Fatal("partial consumption cleared a shortcut or changed the wrong stack")
			}
			if staged {
				committed := s.inventoryOperation(func() {
					if _, err := s.consumeOne(0); err != nil {
						t.Fatal(err)
					}
					s.operation.err = fmt.Errorf("forced rollback after consumption")
				})
				saved, err := db.QuickSlots(s.char.ID)
				if committed || err != nil || len(saved) != 3 || len(s.quickSlots) != 3 || len(s.items) != 1 {
					t.Fatal("aborted consumption changed shortcuts or inventory", saved, s.items, err)
				}
			}
			use()
			saved, err := db.QuickSlots(s.char.ID)
			if err != nil || !slices.Equal(saved, []db.QuickSlot{keep}) || !slices.Equal(s.quickSlots, saved) || len(s.items) != 0 {
				t.Fatalf("depletion left a shortcut: session=%v saved=%v items=%v error=%v", s.quickSlots, saved, s.items, err)
			}
			state := game.New()
			if _, err := state.Apply(waitWorldPacket(t, wire, proto.CharacterSkillBar)); err != nil {
				t.Fatal(err)
			}
			// A bar edit sent before the depletion packet arrived can still
			// contain the old binding. World rejects that empty position.
			s.handleSkillBar(buildCharacterSkillBar([]db.QuickSlot{first, keep, second})[2:])
			if _, err := state.Apply(waitWorldPacket(t, wire, proto.CharacterSkillBar)); err != nil {
				t.Fatal(err)
			}
			replacement := add(db.Item{Bag: item.Bag, Slot: item.Slot, Type: 1, TypeID: 1, Count: 1})
			if _, err := state.Apply(buildAddItem(&replacement)[2:]); err != nil {
				t.Fatal(err)
			}
			if len(state.QuickSlots) != 1 || state.QuickSlots[0].Bag != 100 {
				t.Fatal("replacement inherited a depleted item shortcut", state.QuickSlots)
			}
		})
	}
}
