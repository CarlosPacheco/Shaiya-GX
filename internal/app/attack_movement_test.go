package app

import (
	"github.com/Spelunkern/Shaiya-GX/internal/data/sdata"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"testing"
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
	if !a.attackMovementAllowed() {
		t.Fatal("stationary jump cannot attack")
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
