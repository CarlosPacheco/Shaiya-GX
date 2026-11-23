package app

import (
	"github.com/Spelunkern/Shaiya-GX/internal/data/sdata"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/skilltiming"
	"testing"
)

// tableCadence restores the pure ps_game table for tests of that ceiling.
func tableCadence(t *testing.T) {
	old := skilltiming.AnimationBound
	skilltiming.AnimationBound = false
	t.Cleanup(func() { skilltiming.AnimationBound = old })
}

func TestSharedNativeCadenceAndInstantAckWithoutExtraLatency(t *testing.T) {
	tableCadence(t)
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
	a.clock = 10.84
	if a.prepareSkillMovement(proto.UseSkillPacket(2, 1, true)) || a.skills.lastActionAt != start {
		t.Fatal("alternating skills left earlier than the send-early window")
	}
	a.clock = 10.86 // inside the 150 ms before the 1000 ms deadline; the server holds it until due
	if !a.prepareSkillMovement(proto.UseSkillPacket(2, 1, true)) {
		t.Fatal("Ext. Fast skill was not sent inside the early window")
	}
	if a.wv.attackMoveUntil < a.clock+.799 {
		t.Fatal("movement pause changed")
	}
}
func TestPreparedReleaseDoesNotRestartClientRecovery(t *testing.T) {
	tableCadence(t)
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
	tableCadence(t)
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

func TestJumpKeepsQueuedSkill(t *testing.T) {
	a := inputApp()
	a.state.SelfID = 1
	a.state.Skills[1] = proto.LearnedSkill{ID: 90, Level: 1, Number: 1}
	a.skillDefs = map[sdata.SkillKey]sdata.Skill{{ID: 90, Level: 1}: {Effect: 4, Target: 2}}
	a.skills.queuedSkill = &queuedSkill{packet: proto.UseSkillPacket(1, 1, true), key: sdata.SkillKey{ID: 90, Level: 1}, expires: a.clock + 5}
	a.updateQueuedSkill(&platform.Input{Space: true, Keys: []platform.Key{platform.KeySpace}})
	if a.skills.queuedSkill == nil {
		t.Fatal("jumping dropped the queued skill")
	}
}

// Animation-bound cadence: the table is only a ceiling over the basic-attack
// clip at the weapon-relative animation rate (30 frames at 15-45 fps).
func TestAnimationBoundIntervalFollowsClipAndKeepsTableCeiling(t *testing.T) {
	for _, c := range []struct {
		tier, base uint8
		have       bool
		ms         int64
	}{{9, 5, true, 666}, {5, 5, true, 1000}, {0, 5, true, 2000}, {9, 9, true, 1000}, {9, 0, false, 1000}, {2, 2, true, 1000}} {
		got, ok := skilltiming.IntervalFor(c.tier, c.base, c.have)
		if !ok || got.Milliseconds() != c.ms {
			t.Fatalf("%+v: %v", c, got)
		}
	}
	if _, ok := skilltiming.IntervalFor(255, 5, true); ok {
		t.Fatal("blocked index acted")
	}
}

// The original keeps the action busy until the skill's own animation ends, so
// a skill clip is a floor the send-early window cannot cut.
func TestSkillClipFloorBlocksEarlyRepeat(t *testing.T) {
	a := inputApp()
	a.state.Self.Speed = &proto.CharacterSpeed{Attack: 9}
	a.clock = 10
	a.markCombatAction()
	a.skills.actionFloor = 1
	a.clock = 10.7
	if a.actionReadyWithin(skillSendEarly) {
		t.Fatal("a skill repeated before its 1s clip ended")
	}
	a.clock = 11
	if !a.actionReadyWithin(skillSendEarly) {
		t.Fatal("a skill stayed blocked after its clip ended")
	}
	a.markCombatAction()
	if a.skills.actionFloor != 0 {
		t.Fatal("a basic attack kept the previous skill floor")
	}
}
