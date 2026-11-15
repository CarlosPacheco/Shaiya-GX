package app

import (
	"github.com/Spelunkern/Shaiya-GX/internal/data/sdata"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"testing"
)

func TestPracticalCadenceIsGlobalAndInstantAckDoesNotAddLatency(t *testing.T) {
	a := inputApp()
	a.clock = 10
	a.state.SelfID = 1
	a.state.Skills[1] = proto.LearnedSkill{ID: 90, Level: 1, Number: 1}
	a.state.Skills[2] = proto.LearnedSkill{ID: 91, Level: 1, Number: 2}
	a.skillDefs = map[sdata.SkillKey]sdata.Skill{{ID: 90, Level: 1}: {ResetTime: 1, Effect: 4, Target: 2}, {ID: 91, Level: 1}: {ResetTime: 1, Effect: 4, Target: 2}}
	if !a.prepareSkillMovement(proto.UseSkillPacket(1, 1, true)) {
		t.Fatal("first cast blocked")
	}
	deadline := a.skills.nextCastAt
	a.clock = 10.1
	a.state.LastSkill = proto.SkillResult{Source: 1, Target: 1, ID: 90, Level: 1}
	a.confirmedSkills(proto.SkillUseNoTarget)
	if a.skills.nextCastAt != deadline {
		t.Fatal("instant acknowledgment added network latency to cadence")
	}
	a.clock = 10.299
	if a.learnedSkillPacket(a.state.Skills[2]) != nil {
		t.Fatal("shortcut produced a request during global cadence")
	}
	if a.prepareSkillMovement(proto.UseSkillPacket(2, 1, true)) || a.skills.nextCastAt != deadline {
		t.Fatal("alternating skills bypassed or extended cadence")
	}
	a.clock = 10.3
	if !a.prepareSkillMovement(proto.UseSkillPacket(2, 1, true)) {
		t.Fatal("cadence did not expire at 300ms")
	}
	if a.wv.attackMoveUntil < a.clock+.799 {
		t.Fatal("movement pause was shortened")
	}
}
func TestPreparedReleaseStartsClientCadence(t *testing.T) {
	a := inputApp()
	a.clock = 10
	a.state.SelfID = 1
	a.skillDefs = map[sdata.SkillKey]sdata.Skill{{ID: 90, Level: 1}: {ResetTime: 1, ReadyTime: 1}}
	a.state.LastSkill = proto.SkillResult{Source: 1, Target: 1, ID: 90, Level: 1}
	a.confirmedSkills(proto.SkillUseNoTarget)
	if a.skills.nextCastAt != 10.3 {
		t.Fatal("prepared release omitted cadence")
	}
}
