package world

import (
	"github.com/Spelunkern/Shaiya-GX/internal/mapgeometry"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"math"
	"testing"
	"time"
)

func ladderSession(t *testing.T) (*Session, time.Time) {
	s, now := playerPhysicsFixture(t)
	g := s.movement.geometry
	g.Ladders = []mapgeometry.Ladder{{Bottom: mapgeometry.Position{8, 0, 5}, Top: mapgeometry.Position{8, 8, 5}, Normal: mapgeometry.Position{0, 0, 1}, Width: .7}}
	s.movement.pos = mapgeometry.Position{8, .045, 5.55}
	s.movement.direction = 32768
	s.commitMovement()
	s.handleLadder(proto.LadderPacket(0, mapgeometry.LadderUp))
	if !s.movement.ladder.Active {
		t.Fatal("nearby valid ladder request refused")
	}
	return s, maxTime(now, s.movement.at)
}
func maxTime(a, b time.Time) time.Time {
	if a.After(b) {
		return a
	}
	return b
}
func TestLadderServerOwnsHeightAndPacketFlood(t *testing.T) {
	s, now := ladderSession(t)
	now = now.Add(100 * time.Millisecond)
	s.advancePlayerVertical(now)
	s.commitMovement()
	y := s.movement.pos[1]
	if math.Abs(float64(y-.225)) > .02 {
		t.Fatal("wrong authoritative speed", y)
	}
	for range 100 {
		ok, reason := s.validatePlayerMovement(100, 100, 100, 1, 0, now)
		if !ok || reason == "" {
			t.Fatal("invalid ladder sample not corrected")
		}
	}
	if s.movement.pos[1] != y || s.char.PosX != 8 || s.char.PosZ != 5.55 {
		t.Fatal("packet flood moved climber", s.movement.pos)
	}
	s.movement.ladder.Mode = mapgeometry.LadderHold
	s.advancePlayerVertical(now.Add(500 * time.Millisecond))
	if s.movement.pos[1] != y {
		t.Fatal("hold applied gravity")
	}
	s.advancePlayerVertical(now.Add(2 * time.Second))
	if s.movement.ladder.Active {
		t.Fatal("stale intent remained active")
	}
}
func TestLadderAttackStopAlsoPausesVerticalTravel(t *testing.T) {
	s, now := ladderSession(t)
	s.commitAttackStop(now)
	y := s.movement.pos[1]
	s.advancePlayerVertical(now.Add(799 * time.Millisecond))
	if s.movement.pos[1] != y || s.movement.ladder.Mode != mapgeometry.LadderHold {
		t.Fatal("attack stop did not hold climber")
	}
}
func TestLadderRejectsDistantUnknownAndMalformedRequests(t *testing.T) {
	s, now := playerPhysicsFixture(t)
	s.handleLadder(proto.LadderPacket(999, mapgeometry.LadderUp))
	if s.movement.ladder.Active {
		t.Fatal("unknown ladder accepted")
	}
	s.handleLadder([]byte{0x0d, 0x7f, 0})
	s.handleLadder(proto.LadderPacket(0, 255))
	if s.movement.ladder.Active || s.movement.pos[1] > .15 {
		t.Fatal("invalid ladder request changed physics", now)
	}
}
