package app

import (
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/scene"
)

func TestAbandonAttackClearsChaseAndSwingPause(t *testing.T) {
	a := inputApp()
	a.wv = &worldView{self: &scene.Character{}}
	a.clock = 10
	a.wv.attackSent, a.wv.engagedID, a.wv.approachTarget = true, 7, 7
	a.wv.attackMoveUntil, a.wv.attackPose = 10.5, true
	if !a.abandonAttack() {
		t.Fatal("abandon failed")
	}
	v := a.wv
	if v.attackSent || v.engagedID != 0 || v.approachTarget != 0 || v.attackMoveUntil != 0 || v.attackPose {
		t.Fatalf("attack action survived explicit movement: %+v", v)
	}
}
