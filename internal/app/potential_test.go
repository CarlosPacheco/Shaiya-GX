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
	if packet := a.learnedSkillPacket(proto.LearnedSkill{ID: 24, Level: 3, Number: 1}); packet != nil {
		t.Fatal("a Potential produced a use request")
	}

	// The server announces the reuse when the Potential triggers by itself.
	a.clock = 100
	a.state.LastSkillCooldown = proto.SkillCooldown{Skill: 24, Seconds: 180}
	a.confirmedSkills(proto.GXSkillCooldown)
	if until := a.skills.cooldowns[24]; until != 280 {
		t.Fatalf("reuse ends at %v, want 280", until)
	}
}
