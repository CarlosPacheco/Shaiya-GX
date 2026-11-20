package app

import (
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/data/sdata"
	"github.com/Spelunkern/Shaiya-GX/internal/potential"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
)

// Potentials are learned, draggable to a bar, show a reuse timer the server
// starts, and can never be activated from the client.
func TestPotentialSkillsAreDraggableShowReuseAndCannotBeUsed(t *testing.T) {
	defs := map[sdata.SkillKey]sdata.Skill{
		{ID: 24, Level: 3}: {Name: "Potential Power", Effect: 0, Detail: potential.Detail},
		{ID: 5, Level: 1}:  {Name: "Weapon Power", Effect: 0, Detail: 1},
		{ID: 29, Level: 1}: {Name: "Magic Arrow", Effect: 1, Detail: 21},
	}
	normalizePotentials(defs)
	p := defs[sdata.SkillKey{ID: 24, Level: 3}]
	if p.ResetTime != potential.ReuseSeconds || p.KeepTime != potential.KeepSeconds {
		t.Fatalf("reuse/duration not filled in: %+v", p)
	}
	if defs[sdata.SkillKey{ID: 5, Level: 1}].ResetTime != 0 {
		t.Fatal("an ordinary passive was normalized")
	}
	if !skillDraggable(p) || !skillDraggable(defs[sdata.SkillKey{ID: 29, Level: 1}]) || skillDraggable(defs[sdata.SkillKey{ID: 5, Level: 1}]) {
		t.Fatal("only active skills and Potentials can be dragged to a bar")
	}

	a := inputApp()
	a.skillDefs = defs
	a.state.Skills[7] = proto.LearnedSkill{ID: 24, Level: 3, Number: 7}
	a.state.QuickSlots = []proto.QuickSlot{{Bar: 1, Slot: 3, Bag: 100, Number: 7}}
	if packet := a.actionPacket(13); packet != nil {
		t.Fatal("a Potential produced a use request")
	}

	// The server announces the reuse when the Potential triggers by itself.
	a.clock = 100
	w := proto.NewWriter(proto.GXSkillCooldown)
	w.U16(24)
	w.U16(180)
	w.U16(180)
	if _, err := a.state.Apply(w.Bytes()); err != nil {
		t.Fatal(err)
	}
	a.confirmedSkills(proto.GXSkillCooldown)
	slot := a.skillBar()[13]
	if slot == nil || slot.ID != 24 || a.skillCooldownTotal(slot.ID, p) != 180 {
		t.Fatalf("slot lost the Potential's confirmed timer: %+v", slot)
	}
	if until := a.skills.cooldowns[slot.ID]; until != 280 {
		t.Fatalf("reuse ends at %v, want 280", until)
	}
}

// Timers shortened by a cooldown reduction stay coherent on the bars: a
// running timer drops at once against its original total, and a spell cast
// under a reduction starts with a full bar of the shortened time.
func TestCooldownReductionKeepsTimersCoherent(t *testing.T) {
	a := inputApp()
	a.skillDefs = map[sdata.SkillKey]sdata.Skill{{ID: 31, Level: 1}: {Name: "Spell", Effect: 1, ResetTime: 60}}
	d := a.skillDefs[sdata.SkillKey{ID: 31, Level: 1}]
	a.clock = 100
	a.skills.cooldowns = map[uint16]float64{31: 150} // 60 s spell, 50 s left
	if got := a.skillCooldownTotal(31, d); got != 60 {
		t.Fatalf("total %v, want 60", got)
	}
	a.state.LastSkillCooldown = proto.SkillCooldown{Skill: 31, Seconds: 20} // reduced by 30 s
	a.confirmedSkills(proto.GXSkillCooldown)
	if until := a.skills.cooldowns[31]; until != 120 || a.skillCooldownTotal(31, d) != 60 {
		t.Fatalf("reduced timer ends at %v with total %v, want 120 and 60", until, a.skillCooldownTotal(31, d))
	}
	// A new cast under a reduction: 30 s of 30 s.
	a.state.LastSkillCooldown = proto.SkillCooldown{Skill: 31, Seconds: 30, Total: 30}
	a.confirmedSkills(proto.GXSkillCooldown)
	if a.skills.cooldowns[31] != 130 || a.skillCooldownTotal(31, d) != 30 {
		t.Fatal("a shortened new timer must start with a full bar of its own length")
	}
	// A reduction that covers it clears the timer and its total.
	a.state.LastSkillCooldown = proto.SkillCooldown{Skill: 31}
	a.confirmedSkills(proto.GXSkillCooldown)
	if a.skills.cooldowns[31] > a.clock || a.skillCooldownTotal(31, d) != 60 {
		t.Fatal("a covered timer must be ready and fall back to the ordinary total")
	}
}
