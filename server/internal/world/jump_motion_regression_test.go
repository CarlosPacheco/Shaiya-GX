package world

import (
	"github.com/Spelunkern/Shaiya-GX/internal/collision"
	"github.com/Spelunkern/Shaiya-GX/internal/mapgeometry"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"math"
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

// The client predicts a jump one 60 Hz frame at a time, horizontal then
// vertical. World must reach the same place when a jump crosses a ledge: with
// all the vertical time integrated first, the player lands beside the ledge
// and the horizontal part is then blocked against its face.
func TestJumpAcrossLedgeMatchesFrameByFramePrediction(t *testing.T) {
	s, now := playerPhysicsFixture(t)
	g := s.movement.geometry
	for _, tri := range collision.BlockerBox(collision.Vec{5.2, 0, 0}, collision.Vec{8, 1.0, 30}, 0) {
		g.Add(tri)
	}
	start := mapgeometry.Position{4.85, .045, 5}
	s.movement.pos = start
	s.char.PosX, s.char.PosY, s.char.PosZ = start[0], start[1], start[2]
	s.commitMovement()
	now = now.Add(100 * time.Millisecond)
	if ok, why := s.validatePlayerMovement(start[0], start[1], start[2], 1, 0, now); !ok || why != "" {
		t.Fatal("setup rejected", why)
	}
	if !s.startMovementJump(now) {
		t.Fatal("jump rejected")
	}
	// Client prediction: 24 frames at 4.5 units per second.
	p, fall, dt := start, float32(-2.98), float32(1.0/60)
	frames := 18
	for range frames {
		p = g.GroundMoveDT(p, mapgeometry.Position{4.5 * dt, 0, 0}, dt, true)
		fall = max(-20, fall*(1+2.8*dt))
		p, _ = g.MoveFeet(p, mapgeometry.Position{0, (9 + fall) * dt, 0})
	}
	if p[0] < 5.3 || p[1] < .9 {
		t.Fatal("fixture: the predicted jump did not climb onto the ledge", p)
	}
	now = now.Add(time.Duration(float64(frames) / 60 * float64(time.Second)))
	ok, why := s.validatePlayerMovement(p[0], p[1], p[2], 1, 0, now)
	got := s.movement.pos
	if !ok || why != "" || math.Hypot(float64(got[0]-p[0]), float64(got[2]-p[2])) > .12 || math.Abs(float64(got[1]-p[1])) > .1 {
		t.Fatalf("World left the predicted path: predicted %v, World %v (%s)", p, got, why)
	}
}

// A ceiling ends the jump impulse; the fall restarts at -2.98 instead of
// decelerating from zero (0x43bfa0 clears the impulse on any vertical contact).
func TestCeilingEndsTheImpulseAndRestartsTheFall(t *testing.T) {
	s, now := playerPhysicsFixture(t)
	g := s.movement.geometry
	for _, tri := range []collision.Triangle{{{0, 2.2, 0}, {30, 2.2, 30}, {0, 2.2, 30}}, {{0, 2.2, 0}, {30, 2.2, 0}, {30, 2.2, 30}}} {
		g.Add(tri)
	}
	if !s.startMovementJump(now) {
		t.Fatal("jump rejected")
	}
	for range 60 {
		now = now.Add(time.Second / 60)
		s.advancePlayerVertical(now)
		if !s.movement.jumping {
			break
		}
	}
	if s.movement.jumping || s.movement.fall != -2.98 && s.movement.fall > -2.98 {
		t.Fatalf("the ceiling did not end the impulse: jumping=%v fall=%v pos=%v", s.movement.jumping, s.movement.fall, s.movement.pos)
	}
	if s.movement.pos[1] < .3 {
		t.Fatal("the jump never reached the ceiling", s.movement.pos)
	}
}

// A running jump up a slope keeps its impulse. Terrain that rises under the
// jumper is floor contact, not a ceiling, and must not end the jump: a server
// that landed it ~100 ms in rewound the client and cut the jump short.
func TestRunningJumpUphillKeepsItsImpulse(t *testing.T) {
	s, now := playerPhysicsFixture(t)
	g := &mapgeometry.Geometry{MapSize: 128, Heights: make([]uint16, 65*65)}
	g.Reindex()
	for z := 0; z < 65; z++ {
		for x := 0; x < 65; x++ {
			g.Heights[z*65+x] = uint16(10000 + x*45)
		}
	}
	s.char.PosX, s.char.PosZ = 30, 30
	s.char.PosY = g.Height(30, 30)
	s.movement = playerMovement{geometry: g, mapID: s.char.MapID}
	if err := s.initPlayerMovement(now); err != nil {
		t.Fatal(err)
	}
	slope := (g.Height(31, 30) - g.Height(30, 30))
	t.Logf("slope %.3f height per unit", slope)
	if !s.startMovementJump(now) {
		t.Fatal("jump refused")
	}
	// The 10 Hz position samples leave the server up to ~100 ms behind the
	// client when it accepts the jump.
	x, fall, y := float32(30)+.7, float32(-2.98), s.movement.pos[1]
	for frame := 1; frame <= 24; frame++ { // the first 400 ms, 60 Hz client frames
		dt := float32(1.0 / 60)
		x += 7 * dt
		fall = max(float32(-20), fall*(1+2.8*dt))
		y = max(y+(9+fall)*dt, g.Height(x, 30))
		now = now.Add(time.Second / 60)
		if frame%6 == 0 {
			_, reason := s.validatePlayerMovement(x, y, 30, 0x84, 0, now)
			if !s.movement.jumping {
				t.Fatalf("frame %d: server ended the jump on an uphill slope (reason=%q) y=%.3f server=%v", frame, reason, y, s.movement.pos)
			}
		}
	}
}
