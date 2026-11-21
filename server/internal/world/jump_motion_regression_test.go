package world

import (
	"github.com/Spelunkern/Shaiya-GX/internal/collision"
	"github.com/Spelunkern/Shaiya-GX/internal/mapgeometry"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"testing"
	"time"
)

func TestMovingJumpPredictionMatchesServer(t *testing.T) {
	s, now := playerPhysicsFixture(t)
	p := s.movement.pos
	if !s.startMovementJump(now) {
		t.Fatal("jump refused")
	}
	fall := float32(-2.98)
	peak := p[1]
	for frame := 1; frame <= 100; frame++ {
		dt := float32(1.0 / 60)
		p = s.movement.geometry.GroundMove(p, mapgeometry.Position{0, 0, 5 * dt}, true)
		fall = max(float32(-20), fall*(1+2.8*dt))
		next, _ := s.movement.geometry.MoveFeet(p, mapgeometry.Position{0, (9 + fall) * dt, 0})
		p = next
		peak = max(peak, p[1])
		now = now.Add(time.Second / 60)
		if frame%6 == 0 {
			_, reason := s.validatePlayerMovement(p[0], p[1], p[2], 0x84, 0, now)
			if frame < 40 && (!s.movement.jumping || reason != "") {
				t.Fatalf("frame %d prediction=%v server=%v jumping=%v reason=%s peak=%g", frame, p, s.movement.pos, s.movement.jumping, reason, peak)
			}
		}
	}
}

func TestActiveLandingDoesNotRewindLatestPredictedHorizontalMovement(t *testing.T) {
	for _, idle := range []bool{false, true} {
		fixture, now := playerPhysicsFixture(t)
		s, wire := gmWireSession(t, *fixture.char, false)
		s.movement = fixture.movement
		s.player = RegisterPlayer(*s.char, nil, false, 0, s.out, nil)
		t.Cleanup(s.player.Unregister)
		s.movement.jumping = true
		s.movement.jumpFall = -20
		s.movement.packetAt = now
		s.movement.motion = 0x84
		if idle {
			s.movement.packetAt = now.Add(-time.Second)
		}
		s.tickPlayerMovement(now.Add(50 * time.Millisecond))
		if s.movement.jumping {
			t.Fatal("landing not authoritative")
		}
		if idle {
			select {
			case p := <-wire:
				if proto.Op(p) != proto.GXPlayerCorrection {
					t.Fatal("unexpected idle frame", p)
				}
			case <-time.After(time.Second):
				t.Fatal("idle landing lost server correction")
			}
		} else {
			select {
			case p := <-wire:
				t.Fatal("active landing rewound client", p)
			case <-time.After(20 * time.Millisecond):
			}
		}
	}
}

func TestJumpImpulseStartsAtLatestValidatedStep(t *testing.T) {
	s, now := playerPhysicsFixture(t)
	for _, tri := range collision.BlockerBox(collision.Vec{5, 0, 5.6}, collision.Vec{9, .4, 10}, 0) {
		s.movement.geometry.Add(tri)
	}
	// Walk it the way the client predicts: one 60 Hz frame at a time.
	pending := s.movement.pos
	frame := s.movementSpeed() / 60
	for range 10 {
		pending = s.movement.geometry.GroundMoveDT(pending, mapgeometry.Position{0, 0, frame}, 1.0/60, false)
	}
	if pending[1] < .3 {
		t.Fatal("fixture did not climb step", pending)
	}
	now = now.Add(160 * time.Millisecond)
	if ok, why := s.validatePlayerMovement(pending[0], pending[1], pending[2], 1, 0, now); !ok || why != "" {
		t.Fatal("pre-jump movement rejected", why)
	}
	base := s.char.PosY
	if !s.startMovementJump(now) {
		t.Fatal("jump on new support rejected")
	}
	s.advancePlayerVertical(now.Add(100 * time.Millisecond))
	if s.movement.pos[1] < base+.4 {
		t.Fatal("jump impulse used previous ground height", base, s.movement.pos)
	}
}
