package world

import (
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"testing"
	"time"
)

func TestAttackStopBlocksMovementButNotJumpGravity(t *testing.T) {
	s, now := playerPhysicsFixture(t)
	s.commitAttackStop(now)
	x, z := s.char.PosX, s.char.PosZ
	if !s.attackMoveUntil.Equal(now.Add(800 * time.Millisecond)) {
		t.Fatal("attack pause must be 800ms")
	}
	for i := 1; i <= 4; i++ {
		at := now.Add(time.Duration(i) * 100 * time.Millisecond)
		if ok, _ := s.validatePlayerMovement(x+1, s.char.PosY, z, 1, 0, at); !ok {
			t.Fatal("physics failed")
		}
		if s.char.PosX != x || s.char.PosZ != z {
			t.Fatal("movement during attack pause accepted")
		}
	}
	if !s.startMovementJump(now.Add(410 * time.Millisecond)) {
		t.Fatal("attack stop prevented stationary jump")
	}
	oldY := s.char.PosY
	_, _ = s.validatePlayerMovement(x+1, oldY, z, 4|proto.MotionFlag80, 0, now.Add(450*time.Millisecond))
	if s.char.PosX != x || s.char.PosY <= oldY {
		t.Fatal("attack stop froze vertical motion or allowed XZ")
	}
	if s.attackMovementAllowed() {
		t.Fatal("stationary jump allowed an airborne attack")
	}
	s.movement.credit = 1
	_, _ = s.validatePlayerMovement(x+.1, s.char.PosY, z, 4|proto.MotionFlag80, 0, now.Add(900*time.Millisecond))
	if s.char.PosX <= x {
		t.Fatal("movement did not resume after recovery")
	}
	if s.attackMovementAllowed() {
		t.Fatal("moving jumping attacker allowed")
	}
	// A stationary motion sample cannot authorize a hit before landing.
	_, _ = s.validatePlayerMovement(s.char.PosX, s.char.PosY, s.char.PosZ, 4|proto.MotionFlag80, 0, now.Add(910*time.Millisecond))
	if s.attackMovementAllowed() {
		t.Fatal("stationary sample bypassed airborne restriction")
	}
	s.advancePlayerVertical(now.Add(1600 * time.Millisecond))
	if !s.attackMovementAllowed() {
		t.Fatal("landing did not allow a new attack")
	}
}

func TestAttackStopCannotBankMovementCredit(t *testing.T) {
	s, now := playerPhysicsFixture(t)
	s.commitAttackStop(now)
	x, z := s.char.PosX, s.char.PosZ
	// No samples during the pause: only 100ms after recovery can earn credit.
	_, _ = s.validatePlayerMovement(x+2, s.char.PosY, z, 1, 0, now.Add(900*time.Millisecond))
	if s.char.PosX-x > .51 {
		t.Fatal("attack recovery time banked movement credit", s.char.PosX-x)
	}
}
