package app

import "testing"

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
	if a.wv.attackMoveUntil != 10.5 {
		t.Fatal("pause differs from 500ms", a.wv.attackMoveUntil)
	}
}
