package world

import (
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/Spelunkern/Shaiya-GX/internal/game"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/server/internal/db"
)

func TestGMSetLevelPreservesEquipmentAndSkills(t *testing.T) {
	fixture, defs := lapisSession(t)
	ch := fixture.char
	defs.Item(1, 1).ReqLevel = 50
	ch.Level, ch.StatPoint, ch.SkillPoint = 20, 19, 18
	if err := db.SaveCharacter(ch); err != nil {
		t.Fatal(err)
	}
	items := fixture.items
	if err := db.AddCharacterItem(ch.ID, &items[0]); err != nil {
		t.Fatal(err)
	}
	skill := db.CharacterSkill{SkillID: 125, Number: 1, Level: 1}
	if err := db.AddCharacterSkill(ch.ID, skill); err != nil {
		t.Fatal(err)
	}
	s, wire := gmWireSession(t, *ch, true)
	s.items, s.skills = items, []db.CharacterSkill{skill}
	initialItems, err := db.CharacterItems(ch.ID)
	if err != nil || len(initialItems) != 1 {
		t.Fatalf("missing equipped item fixture: %v (%v)", initialItems, err)
	}

	state := game.New()
	state.SelfID, state.Phase = ch.ID, game.PhaseInWorld
	applyUntilReply := func() {
		t.Helper()
		timeout := time.NewTimer(time.Second)
		defer timeout.Stop()
		for {
			select {
			case p := <-wire:
				if proto.Op(p) == proto.LevelUpSelf {
					t.Fatal("administrative level change announced a gameplay level-up")
				}
				if _, err := state.Apply(p); err != nil {
					t.Fatal(err)
				}
				if proto.Op(p) == proto.GXGMState {
					return
				}
			case <-timeout.C:
				t.Fatal("GM did not reply")
			}
		}
	}
	for _, level := range []uint16{1, defs.LevelCap} {
		s.handleGMCommand(fmt.Sprintf("/set level %d", level))
		applyUntilReply()
		saved, err := db.CharacterByID(ch.ID)
		if err != nil {
			t.Fatal(err)
		}
		if saved.Level != level || saved.Exp != defs.Exp[level-1] || saved.Str != ch.Str ||
			saved.StatPoint != ch.StatPoint || saved.SkillPoint != ch.SkillPoint ||
			state.Slots[ch.Slot].Level != level || state.Details.Exp != saved.Exp ||
			state.Slots[ch.Slot].EquipType[weaponSlot] != items[0].Type {
			t.Fatalf("inconsistent level or character data: level %d, EXP %d, client %d/%d", saved.Level, saved.Exp, state.Slots[ch.Slot].Level, state.Details.Exp)
		}
	}
	for _, text := range []string{"/set level 0", "/set level -1", "/set level 1.5", fmt.Sprintf("/set level %d", defs.LevelCap+1)} {
		s.handleGMCommand(text)
		applyUntilReply()
		if s.char.Level != defs.LevelCap {
			t.Fatalf("invalid level applied for %q", text)
		}
	}
	s.isAdmin = false
	s.handleGMCommand("/set level 1")
	applyUntilReply()
	if s.char.Level != defs.LevelCap {
		t.Fatal("non-GM changed level")
	}
	savedItems, err := db.CharacterItems(ch.ID)
	if err != nil || !reflect.DeepEqual(savedItems, initialItems) {
		t.Fatalf("equipped items changed: %v (%v)", savedItems, err)
	}
	savedSkills, err := db.CharacterSkills(ch.ID)
	if err != nil || len(savedSkills) != 1 || savedSkills[0] != skill {
		t.Fatalf("learned skills changed: %v (%v)", savedSkills, err)
	}
}
