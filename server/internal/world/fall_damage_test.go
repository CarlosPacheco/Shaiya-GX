package world

import (
	"github.com/Spelunkern/Shaiya-GX/internal/collision"
	"github.com/Spelunkern/Shaiya-GX/internal/mapgeometry"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"math"
	"testing"
	"time"
)

func TestFallDamageCurve(t *testing.T) {
	for _, tc := range []struct {
		drop float32
		want uint32
	}{{0, 0}, {6, 0}, {12, 50}, {18, 200}, {24, 450}, {30, 800}, {42, 1600}, {80, 1600}, {float32(math.NaN()), 0}, {float32(math.Inf(1)), 0}} {
		if got := fallDamage(tc.drop, 1600); got != tc.want {
			t.Fatal(tc, got)
		}
	}
}
func fallSession(t *testing.T, height float32) (*Session, time.Time) {
	t.Helper()
	t.Setenv("RAPTURE_FALL_DAMAGE", "true")
	s, _ := forgeRuleSession(t)
	g := &mapgeometry.Geometry{}
	g.Reindex()
	g.Add(collision.Triangle{{0, 0, 0}, {0, 0, 30}, {30, 0, 0}})
	g.Add(collision.Triangle{{30, 0, 30}, {30, 0, 0}, {0, 0, 30}})
	g.Add(collision.Triangle{{0, float64(height), 0}, {0, float64(height), 20}, {9, float64(height), 0}})
	g.Add(collision.Triangle{{9, float64(height), 20}, {9, float64(height), 0}, {0, float64(height), 20}})
	s.char.MapID, s.char.PosX, s.char.PosY, s.char.PosZ = 60045, 8, height+.05, 5
	s.char.CurrentHP = s.stats().MaxHP
	s.movement = playerMovement{geometry: g, mapID: s.char.MapID}
	now := time.Now()
	if err := s.initPlayerMovement(now); err != nil {
		t.Fatal(err)
	}
	if !s.movement.fallReady {
		t.Fatal("fixture did not begin on ledge")
	}
	// Move off an authored ledge. Subsequent height is integrated by the server.
	s.movement.pos[0] = 12
	s.commitMovement()
	return s, now
}
func finishTestFall(s *Session, now time.Time) {
	for i := 1; i <= 600; i++ {
		s.advancePlayerVertical(now.Add(time.Duration(i) * time.Second / 120))
		s.commitMovement()
	}
}
func TestAuthoritativeLandingDamagesOnceAndCanKill(t *testing.T) {
	for _, height := range []float32{4, 12, 42} {
		t.Run(string(rune('A'+int(height))), func(t *testing.T) {
			s, now := fallSession(t, height)
			before := s.char.CurrentHP
			maxHP := s.stats().MaxHP
			finishTestFall(s, now)
			expected := min(before, fallDamage(height, maxHP))
			if s.char.CurrentHP != before-expected {
				t.Fatal("landing damage mismatch", height, before, s.char.CurrentHP, expected, s.movement.pos)
			}
			remaining := s.char.CurrentHP
			s.advancePlayerVertical(now.Add(6 * time.Second))
			if s.char.CurrentHP != remaining {
				t.Fatal("landing applied twice")
			}
			if height >= 40 && (!s.dead || s.deathLoss != -1 || len(s.damageLedger) != 0) {
				t.Fatal("environmental lethal fall not handled safely")
			}
		})
	}
}
func TestSpawnAndImmuneFallsDoNotDamage(t *testing.T) {
	for _, spawn := range []bool{false, true} {
		name := "immune"
		if spawn {
			name = "spawn"
		}
		t.Run(name, func(t *testing.T) {
			s, now := fallSession(t, 20)
			before := s.char.CurrentHP
			if spawn {
				s.movement.fallReady = false
			} else {
				s.isAdmin, s.gmImmune = true, true
			}
			finishTestFall(s, now)
			if s.char.CurrentHP != before {
				t.Fatal("spawn or immunity exemption failed", spawn)
			}
		})
	}
}
func TestDeepWaterCancelsTrackedFall(t *testing.T) {
	s, now := fallSession(t, 20)
	before := s.char.CurrentHP
	g := s.movement.geometry
	g.Triangles = nil
	g.Reindex()
	g.Water = true
	g.MapSize = 32
	g.Heights = make([]uint16, 17*17)
	for i := range g.Heights {
		g.Heights[i] = 9500
	}
	finishTestFall(s, now)
	if s.char.CurrentHP != before || s.movement.fallTracking || s.movement.pos[1] < -1.601 {
		t.Fatal("water landing did not clear fall safely", s.char.CurrentHP, s.movement)
	}
}

func TestFallLandingSendsFramedDamageThenAuthoritativePools(t *testing.T) {
	s, now := fallSession(t, 12)
	wire, packets := gmWireSession(t, *s.char, false)
	s.out = wire.out
	finishTestFall(s, now)
	timer := time.NewTimer(time.Second)
	defer timer.Stop()
	seen := false
	for {
		select {
		case p := <-packets:
			if proto.Op(p) == proto.GXFallDamage {
				v, err := proto.DecodeFallDamage(p)
				if err != nil || v.Target != s.char.ID || v.Amount == 0 {
					t.Fatal("bad actual fall frame", v, err)
				}
				if seen {
					t.Fatal("duplicate landing notification")
				}
				seen = true
			}
			if proto.Op(p) == proto.CharacterCurrentHitpoints {
				hp, err := proto.DecodePools(p)
				if err != nil || hp.HP != s.stats().CurrentHP || !seen {
					t.Fatal("missing or unordered authoritative hp", hp, err)
				}
				return
			}
		case <-timer.C:
			t.Fatal("fall notification or hp frame missing")
		}
	}
}

func TestFallDamageExplicitlyDisabled(t *testing.T) {
	s, now := fallSession(t, 32)
	t.Setenv("RAPTURE_FALL_DAMAGE", "false")
	before := s.char.CurrentHP
	finishTestFall(s, now)
	if s.char.CurrentHP != before || s.dead {
		t.Fatal("disabled fall damage changed health or killed player")
	}
}

func TestGMTurboFallKeepsGravityAndHealth(t *testing.T) {
	for _, tc := range []struct {
		name         string
		admin, turbo bool
		survives     bool
	}{
		{"GM in turbo", true, true, true},
		{"GM with turbo off", true, false, false},
		{"ordinary player", false, true, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s, now := fallSession(t, 42)
			before := s.char.CurrentHP
			s.isAdmin, s.gmTurbo = tc.admin, tc.turbo
			finishTestFall(s, now)
			if s.char.PosY > .13 || s.movement.fallTracking {
				t.Fatal("gravity or landing changed", s.movement.pos)
			}
			if tc.survives {
				if s.dead || s.char.CurrentHP != before {
					t.Fatal("turbo GM took fall damage")
				}
			} else if !s.dead {
				t.Fatal("normal lethal fall was ignored")
			}
		})
	}
}
