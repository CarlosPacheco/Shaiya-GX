package app

import (
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/data/sdata"
	"github.com/Spelunkern/Shaiya-GX/internal/game"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
)

func TestAttackerJumpMovementAndSharedPause(t *testing.T) {
	a := inputApp()
	a.clock = 10
	a.wv.jumping = true
	a.wv.moving = true
	if a.attackMovementAllowed() {
		t.Fatal("moving jump can attack")
	}
	a.wv.moving = false
	if a.attackMovementAllowed() {
		t.Fatal("stationary jump can attack before landing")
	}
	a.wv.jumping = false
	a.wv.moving = true
	if !a.attackMovementAllowed() {
		t.Fatal("grounded moving player cannot stop to attack")
	}
	a.wv.moving = false
	if !a.stopForAttack() {
		t.Fatal("stop failed")
	}
	if a.wv.attackMoveUntil != 10.8 {
		t.Fatal("pause differs from 800ms", a.wv.attackMoveUntil)
	}
}

func TestJumpBuffersOnlyStationaryCombatOrders(t *testing.T) {
	a := inputApp()
	const target = 42
	a.state.Target = target
	a.state.Entities[target] = &game.Entity{ID: target, Kind: game.KindMob}
	a.wv.startJumpAnimation()
	if !a.attackTarget() || a.wv.landingAction.target != target || a.wv.approachTarget != 0 {
		t.Fatal("basic attack was not held during takeoff")
	}
	a.state.Skills[1] = proto.LearnedSkill{ID: 77, Level: 1, Number: 1}
	if !a.approachSkillPacket(proto.UseSkillPacket(1, target, false)) || a.wv.landingAction.skill.ID != 77 {
		t.Fatal("spell did not replace the basic attack in the landing queue")
	}
	a.updateLandingAction(&platform.Input{Space: true})
	if a.wv.landingAction.skill.ID != 77 {
		t.Fatal("holding Jump discarded the order")
	}
	a.wv.jumpMoved = true
	a.updateLandingAction(&platform.Input{})
	if a.wv.landingAction.target != 0 || a.wv.landingAction.packet != nil || !a.attackTarget() || a.wv.landingAction.target != 0 {
		t.Fatal("moving jump retained or acquired an order")
	}
	a.wv.jumpMoved = false
	a.attackTarget()
	a.state.Target = target + 1
	a.updateLandingAction(&platform.Input{})
	if a.wv.landingAction.target != 0 {
		t.Fatal("retargeting kept a stale attack")
	}
}

func TestEverySkillStopsMovementIncludingSelfCasts(t *testing.T) {
	for _, effect := range []uint8{1, 2, 3, 4, 5} {
		a := inputApp()
		a.clock = 10
		a.state.Skills[1] = proto.LearnedSkill{ID: 77, Level: 1, Number: 1}
		a.skillDefs = map[sdata.SkillKey]sdata.Skill{{ID: 77, Level: 1}: {Effect: effect}}
		if !a.prepareSkillMovement(proto.UseSkillPacket(1, a.state.SelfID, true)) || a.wv.attackMoveUntil != 10.8 {
			t.Fatal("skill did not commit shared pause", effect, a.wv.attackMoveUntil)
		}
	}
}
