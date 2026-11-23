package world

import (
	"testing"
	"time"

	"github.com/Spelunkern/Shaiya-GX/internal/game"
	"github.com/Spelunkern/Shaiya-GX/internal/mapgeometry"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
)

func oxygenFixture(t *testing.T) (*Session, <-chan []byte) {
	t.Helper()
	s, _ := forgeRuleSession(t)
	wire, packets := gmWireSession(t, *s.char, false)
	s.out, s.aesSend, s.expanded = wire.out, wire.aesSend, wire.expanded
	g := &mapgeometry.Geometry{MapSize: 32, Water: true, Heights: make([]uint16, 17*17)}
	for i := range g.Heights {
		g.Heights[i] = 9500
	}
	s.movement = playerMovement{geometry: g, mapID: s.char.MapID, pos: mapgeometry.Position{5, -1.5, 5}}
	s.commitMovement()
	s.char.CurrentHP = s.stats().MaxHP
	return s, packets
}

func TestOxygenRefillsAtSurfaceAndResumesFromRemainingAir(t *testing.T) {
	s, packets := oxygenFixture(t)
	now := time.Now()
	s.tickOxygen(now)
	s.movement.pos[1] = -1.6
	s.tickOxygen(now.Add(time.Second))
	if s.oxygen.phase != proto.OxygenHidden {
		t.Fatal("surface swimming or entering the water used oxygen")
	}
	s.movement.pos[1] = -2
	check := func(at time.Duration, phase proto.OxygenPhase, ms uint16) {
		t.Helper()
		s.tickOxygen(now.Add(at))
		want := proto.OxygenStatus{Phase: phase, RemainingMS: ms}
		got, err := proto.DecodeOxygen(waitWorldPacket(t, packets, proto.GXOxygen))
		if err != nil || got != want {
			t.Fatalf("at %s: oxygen %+v, want %+v, err %v", at, got, want, err)
		}
	}
	check(2*time.Second, proto.OxygenDraining, 30000)
	s.movement.pos[1] = -1.5
	check(22*time.Second, proto.OxygenRefilling, 10000)
	s.movement.pos[1] = -2
	check(23*time.Second, proto.OxygenDraining, 20000)
	s.movement.pos[1] = -1.5
	check(24*time.Second, proto.OxygenRefilling, 19000)
	check(26*time.Second, proto.OxygenHidden, 0)
}

func TestDrowningUsesAuthoritativeHealthAndEnvironmentalDeath(t *testing.T) {
	s, packets := oxygenFixture(t)
	t.Setenv("RAPTURE_FALL_DAMAGE", "false")
	now := time.Now()
	full := s.stats().MaxHP
	s.movement.pos[1] = -2
	s.tickOxygen(now)
	client := game.New()
	client.SelfID = s.char.ID
	apply := func(op proto.Opcode) {
		t.Helper()
		if _, err := client.Apply(waitWorldPacket(t, packets, op)); err != nil {
			t.Fatalf("packet %#x: %v", op, err)
		}
	}
	apply(proto.GXOxygen)
	s.tickOxygen(now.Add(32*time.Second - time.Millisecond))
	if s.char.CurrentHP != full || s.oxygen.remaining != 0 {
		t.Fatal("drowning started before 30 seconds plus the first damage interval")
	}
	for i := 1; i <= 5; i++ {
		before := s.char.CurrentHP
		s.tickOxygen(now.Add(30*time.Second + time.Duration(i)*drowningInterval))
		apply(proto.GXDrowningDamage)
		apply(proto.CharacterCurrentHitpoints)
		want := min(before, (full+4)/5)
		if client.LastDrowningDamage.Amount != want || client.Pools.HP != before-want || client.Pools.HP != s.char.CurrentHP {
			t.Fatal("drowning damage or authoritative HP mismatch", i, client.Pools, client.LastDrowningDamage)
		}
	}
	apply(proto.CharacterDeath)
	apply(proto.GXOxygen)
	if !s.dead || s.deathLoss != -1 || !client.Self.Dead || !client.DeathLossFree || client.Oxygen.Phase != proto.OxygenHidden {
		t.Fatal("drowning did not complete a loss-free environmental death")
	}
}

func TestSurfacingBeforeDrowningTickPreventsDamage(t *testing.T) {
	s, packets := oxygenFixture(t)
	now := time.Now()
	s.movement.pos[1] = -2
	s.tickOxygen(now)
	waitWorldPacket(t, packets, proto.GXOxygen)
	full := s.char.CurrentHP
	s.tickOxygen(now.Add(30 * time.Second))
	s.movement.pos[1] = -1.5
	s.tickOxygen(now.Add(32 * time.Second))
	v, err := proto.DecodeOxygen(waitWorldPacket(t, packets, proto.GXOxygen))
	if err != nil || v.Phase != proto.OxygenRefilling || s.char.CurrentHP != full {
		t.Fatal("surfaced character still drowned", v, err, s.char.CurrentHP)
	}
}

// GM "attack on" gives no protection from the water, drowning announces its
// own death and the body floats up to the surface pose.
func TestGMImmunityDoesNotStopDrowningAndTheDrownedBodyFloats(t *testing.T) {
	s, packets := oxygenFixture(t)
	t.Setenv("RAPTURE_FALL_DAMAGE", "false")
	s.isAdmin, s.gmImmune = true, true
	now := time.Now()
	s.movement.pos[1] = -3
	s.char.PosY = -3
	s.tickOxygen(now)
	before := s.char.CurrentHP
	s.tickOxygen(now.Add(30*time.Second + drowningInterval))
	if s.char.CurrentHP >= before {
		t.Fatal("an immune GM did not take drowning damage")
	}
	for i := 2; i <= 6 && !s.dead; i++ {
		s.tickOxygen(now.Add(30*time.Second + time.Duration(i)*drowningInterval))
	}
	if !s.dead {
		t.Fatal("the GM never drowned")
	}
	id, err := proto.DecodeDrowningDeath(waitWorldPacket(t, packets, proto.GXDrowningDeath))
	if err != nil || id != s.char.ID {
		t.Fatalf("drowning death notice: %d, %v", id, err)
	}
	start := s.char.PosY
	if start != -3 {
		t.Fatalf("the body jumped to %.2f instead of rising", start)
	}
	step := now.Add(40 * time.Second)
	s.drownedRiseAt = step
	s.tickDrownedFloat(step.Add(500 * time.Millisecond))
	if got := s.char.PosY; got <= start || got >= drownedFloatY {
		t.Fatalf("half a second of rise left the body at %.2f", got)
	}
	s.tickDrownedFloat(step.Add(5 * time.Second))
	if s.char.PosY != drownedFloatY || s.movement.pos[1] != drownedFloatY || !s.drownedRiseAt.IsZero() {
		t.Fatalf("the drowned body did not settle at the surface: %.2f", s.char.PosY)
	}
}

func TestFallDamageIgnoresAttackOnButNotTurbo(t *testing.T) {
	s, _ := oxygenFixture(t)
	t.Setenv("RAPTURE_FALL_DAMAGE", "true")
	s.isAdmin, s.gmImmune = true, true
	full := s.char.CurrentHP
	s.applyFallDamage(30)
	if s.char.CurrentHP >= full {
		t.Fatal("attack on protected a GM from fall damage")
	}
	s.char.CurrentHP = full
	s.gmTurbo = true
	s.applyFallDamage(30)
	if s.char.CurrentHP != full {
		t.Fatal("turbo no longer cancels fall damage")
	}
}
