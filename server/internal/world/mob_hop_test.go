package world

import (
	"testing"
	"time"

	"github.com/Spelunkern/Shaiya-GX/internal/mapnav"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/server/internal/db"
)

func TestMobMovementUsesOnePositionAndClockForCombatAndPackets(t *testing.T) {
	now := time.Now()
	m := &MapMob{ID: 1, hasCell: true, event: &eventActor{}, spawnHeightKnown: true,
		def: &db.MobDef{NormalStep: 4, NormalTimeMS: 1000, ChaseStep: 7, ChaseTimeMS: 1000}}
	m.setPoint(mapnav.Point{100, 2, 100})
	goal := mapnav.Point{104, 6, 103}
	move := m.moveTo(0, now, goal, mobMotionRun)
	packet, err := proto.DecodeMobMove(move.Move[2:])
	if err != nil || packet.DurationMS != 747 || packet.X != goal[0] || packet.Y != goal[1] || packet.Z != goal[2] {
		t.Fatalf("first move = %+v, %v", packet, err)
	}
	halfway := now.Add(m.moveDuration / 2)
	mid := mapnav.Point{102, 4, 101.5}
	if !m.movingAt(halfway) || m.positionAt(halfway) != mid {
		t.Fatalf("halfway position = %v, want %v", m.positionAt(halfway), mid)
	}
	enter, remaining := m.appearance(halfway, false)
	seen, err := proto.DecodeMobEnter(enter[2:])
	if err != nil || [3]float32{seen.X, seen.Y, seen.Z} != mid {
		t.Fatalf("enter in mid-move = %+v, %v", seen, err)
	}
	if tail, err := proto.DecodeMobMove(remaining[2:]); err != nil || tail.DurationMS != 373 {
		t.Fatalf("remaining movement = %+v, %v", tail, err)
	}
	arrival := now.Add(m.moveDuration)
	if m.movingAt(arrival) || m.positionAt(arrival) != goal {
		t.Fatalf("packet arrival position = %v, want %v", m.positionAt(arrival), goal)
	}
	m.died(halfway)
	stop, err := proto.DecodeMobMove(m.deathPositionPacket()[2:])
	if err != nil || m.point() != mid || stop.DurationMS != 0 || [3]float32{stop.X, stop.Y, stop.Z} != mid {
		t.Fatalf("death position = %v (packet %+v, %v), want %v", m.point(), stop, err, mid)
	}
	move = m.moveTo(0, arrival, mapnav.Point{106, 4, 101.5}, mobMotionWalk)
	packet, err = proto.DecodeMobMove(move.Move[2:])
	if err != nil || packet.DurationMS != 1082 || packet.Motion != mobMotionWalk {
		t.Fatalf("normal walk = %+v, %v", packet, err)
	}
}
