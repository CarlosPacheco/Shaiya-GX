package app

import (
	"bytes"
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/data/sdata"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
)

// A trap goes on the ground where the caster stands: it is sent without a
// victim, even with nobody selected or nearby.
func TestTrapSkillIsSentWithoutATarget(t *testing.T) {
	a := inputApp()
	a.skillDefs = map[sdata.SkillKey]sdata.Skill{{ID: 113, Level: 1}: {Name: "Trap", Effect: 5, Detail: 101, Target: 1, SP: 1, MP: 1}}
	a.state.Target = 0
	got := a.learnedSkillPacket(proto.LearnedSkill{ID: 113, Level: 1, Number: 2})
	if want := proto.UseSkillPacket(2, 0, false); !bytes.Equal(got, want) {
		t.Fatalf("packet %v, want %v", got, want)
	}
}
