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

func TestMobChaseAndReturnUseRunningSpeed(t *testing.T) {
	const mapID = 62082
	now := time.Now()
	m := &MapMob{ID: 1, homeX: 100, homeY: 2, homeZ: 100, event: &eventActor{}, spawnHeightKnown: true,
		def: &db.MobDef{Size: 1, NormalStep: 16, NormalTimeMS: 4000, ChaseStep: 12, ChaseTimeMS: 1800}}
	m.def.Attacks[0].Range = 2
	m.setPoint(mapnav.Point{100, 2, 100})
	target := &PlayerCombatSnapshot{PosX: 103.36, PosY: 2, PosZ: 100}
	mods := m.mods()
	var tick MobCombatTick
	if !m.stepChase(now, mapID, target, m.targetDistance(target), &mods, &tick) {
		t.Fatal("missing chase movement")
	}
	chase, err := proto.DecodeMobMove(tick.Moves[0].Move[2:])
	if err != nil || chase.Motion != mobMotionRun || chase.DurationMS != 26 {
		t.Errorf("0.16-unit chase at 6.367 units/s = %+v, %v", chase, err)
	}
	if m.movingAt(now.Add(26 * time.Millisecond)) {
		t.Error("short chase still active at its arrival time")
	}

	m.setPoint(mapnav.Point{112, 2, 100})
	m.returningHome = true
	if !m.stepHome(now.Add(time.Second), mapID, &mods, &tick) {
		t.Fatal("missing return-home movement")
	}
	home, err := proto.DecodeMobMove(tick.Moves[1].Move[2:])
	if err != nil || home.Motion != mobMotionRun || home.DurationMS != 1885 || home.X != 100 {
		t.Errorf("12-unit return at 6.367 units/s = %+v, %v", home, err)
	}
}

func TestMobRunRetimesFromCurrentPositionWhenSlowed(t *testing.T) {
	now := time.Now()
	m := &MapMob{ID: 1, event: &eventActor{}, spawnHeightKnown: true,
		def: &db.MobDef{ChaseStep: 12, ChaseTimeMS: 1800}}
	origin, goal := mapnav.Point{100, 2, 100}, mapnav.Point{112, 2, 100}
	m.setPoint(origin)
	m.moveTo(0, now, goal, mobMotionRun)
	halfway := now.Add(m.moveDuration / 2)
	mid := mapnav.Point{106, 2, 100}
	m.Effects = []MobEffect{{Def: &db.SkillDef{TypeDetail: detailAbilityDown, AbilityType: [3]uint8{17}, AbilityValue: [3]int16{2}}}}
	var tick MobCombatTick
	m.refreshMoveSpeed(0, halfway, &tick)
	if len(tick.Moves) != 1 || m.positionAt(halfway) != mid {
		t.Fatalf("slow changed position or missed the update: %v (%d packets)", m.positionAt(halfway), len(tick.Moves))
	}
	move, err := proto.DecodeMobMove(tick.Moves[0].Move[2:])
	if err != nil || move.Motion != mobMotionRun || move.X != goal[0] || move.DurationMS != 1448 {
		t.Fatalf("6-unit run with tier 0 at 4.144 units/s = %+v, %v", move, err)
	}
	m.refreshMoveSpeed(0, halfway, &tick)
	if len(tick.Moves) != 1 {
		t.Fatal("unchanged speed sent another movement")
	}
	cleared := halfway.Add(724 * time.Millisecond)
	pos := m.positionAt(cleared)
	m.Effects = nil
	m.refreshMoveSpeed(0, cleared, &tick)
	if len(tick.Moves) != 2 || m.positionAt(cleared) != pos {
		t.Fatalf("clearing the slow changed position or missed the update: %v (%d packets)", m.positionAt(cleared), len(tick.Moves))
	}
	move, err = proto.DecodeMobMove(tick.Moves[1].Move[2:])
	if err != nil || move.Motion != mobMotionRun || move.X != goal[0] || move.DurationMS != 472 {
		t.Fatalf("3-unit run at restored speed = %+v, %v", move, err)
	}
	if m.movingAt(cleared.Add(472*time.Millisecond)) || m.positionAt(cleared.Add(472*time.Millisecond)) != goal {
		t.Fatal("restored movement did not arrive on time")
	}
}
