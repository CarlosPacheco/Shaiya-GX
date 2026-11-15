package app

import (
	"github.com/Spelunkern/Shaiya-GX/internal/data/sdata"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"testing"
)

func TestSharedNativeCadenceAndInstantAckWithoutExtraLatency(t *testing.T) {
	a := inputApp()
	a.clock = 10
	a.state.SelfID = 1
	a.state.Self.Speed = &proto.CharacterSpeed{Attack: 9}
	a.state.Skills[1] = proto.LearnedSkill{ID: 90, Level: 1, Number: 1}
	a.state.Skills[2] = proto.LearnedSkill{ID: 91, Level: 1, Number: 2}
	a.skillDefs = map[sdata.SkillKey]sdata.Skill{{ID: 90, Level: 1}: {ResetTime: 1, Effect: 4, Target: 2}, {ID: 91, Level: 1}: {ResetTime: 1, Effect: 4, Target: 2}}
	if !a.prepareSkillMovement(proto.UseSkillPacket(1, 1, true)) {
		t.Fatal("first cast blocked")
	}
	start := a.skills.lastActionAt
	a.clock = 10.1
	a.state.LastSkill = proto.SkillResult{Source: 1, Target: 1, ID: 90, Level: 1}
	a.confirmedSkills(proto.SkillUseNoTarget)
	if a.skills.lastActionAt != start {
		t.Fatal("acknowledgment added latency")
	}
	a.clock = 10.999
	if a.prepareSkillMovement(proto.UseSkillPacket(2, 1, true)) || a.skills.lastActionAt != start {
		t.Fatal("alternating skills bypassed native speed cap")
	}
	a.clock = 11
	if !a.prepareSkillMovement(proto.UseSkillPacket(2, 1, true)) {
		t.Fatal("Ext. Fast did not expire at 1000ms")
	}
	if a.wv.attackMoveUntil < a.clock+.799 {
		t.Fatal("movement pause changed")
	}
}
func TestPreparedReleaseDoesNotRestartClientRecovery(t *testing.T) {
	a := inputApp()
	a.state.SelfID = 1
	a.state.Self.Speed = &proto.CharacterSpeed{Attack: 9}
	a.clock = 10
	a.markCombatAction()
	a.skillDefs = map[sdata.SkillKey]sdata.Skill{{ID: 90, Level: 1}: {ResetTime: 1, ReadyTime: 4}}
	a.clock = 11.5
	a.state.LastSkill = proto.SkillResult{Source: 1, Target: 1, ID: 90, Level: 1}
	a.confirmedSkills(proto.SkillUseNoTarget)
	if a.skills.lastActionAt != 10 || !a.combatActionReady() {
		t.Fatal("prepared release charged a second interval")
	}
}
func TestBasicFeedbackAndLiveSpeedShareSkillRecovery(t *testing.T) {
	a := inputApp()
	a.state.SelfID = 1
	a.clock = 10
	a.state.Self.Speed = &proto.CharacterSpeed{Attack: 5}
	a.state.LastAttack = proto.AttackResult{Source: 1, Target: 2}
	a.confirmedAnimation(proto.NewWriter(proto.CharacterUserAutoAttack).Bytes())
	a.clock = 11
	if a.combatActionReady() {
		t.Fatal("basic attack did not reserve spell recovery")
	}
	a.state.Self.Speed.Attack = 9
	if !a.combatActionReady() {
		t.Fatal("live haste ignored")
	}
	a.state.Self.Speed.Attack = 255
	a.clock = 100
	if a.combatActionReady() {
		t.Fatal("blocked speed allowed skill")
	}
}

func TestSharedRecoveryBuffersOnlyLatestSkillAndMovementCancels(t *testing.T) {
	a := inputApp()
	a.state.SelfID = 1
	a.state.Self.Speed = &proto.CharacterSpeed{Attack: 9}
	a.clock = 10
	a.markCombatAction()
	a.state.Skills[1] = proto.LearnedSkill{ID: 90, Level: 1, Number: 1}
	a.state.Skills[2] = proto.LearnedSkill{ID: 91, Level: 1, Number: 2}
	a.skillDefs = map[sdata.SkillKey]sdata.Skill{{ID: 90, Level: 1}: {Effect: 4, Target: 2}, {ID: 91, Level: 1}: {Effect: 4, Target: 2}}
	a.clock = 10.5
	if a.prepareSkillMovement(proto.UseSkillPacket(1, 1, true)) || a.skills.queuedSkill == nil {
		t.Fatal("recovery did not buffer spell")
	}
	a.prepareSkillMovement(proto.UseSkillPacket(2, 1, true))
	if a.skills.queuedSkill.key.ID != 91 || a.skills.lastActionAt != 10 {
		t.Fatal("queue stacked or restarted recovery")
	}
	a.updateQueuedSkill(&platform.Input{Down: "W"})
	if a.skills.queuedSkill != nil {
		t.Fatal("manual movement did not cancel queued spell")
	}
}
