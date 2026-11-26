//go:build integration

package app

import (
	"math"
	"os"
	"runtime"
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/data/archive"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
	"github.com/Spelunkern/Shaiya-GX/internal/scene"
)

func TestMobMovementAndAttackContactGL(t *testing.T) {
	if os.Getenv("SHAIYA_TEST_RENDER") != "1" {
		t.Skip("requires original assets and OpenGL")
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	win, err := platform.Open(platform.Config{Width: 64, Height: 64, Hidden: true})
	if err != nil {
		t.Fatal(err)
	}
	defer win.Close()
	if err := render.Init(false); err != nil {
		t.Fatal(err)
	}
	arc, err := archive.OpenDirectory(os.Getenv("SHAIYA_DATA"))
	if err != nil {
		t.Fatal(err)
	}
	defer arc.Close()
	assets := scene.NewAssets(arc)
	const playerID, mobID = 1, 77
	newApp := func(template uint16) *App {
		a := inputApp()
		a.assets = assets
		a.clock, a.frameDT = 10, 1.0/60
		a.state.SelfID = playerID
		spawn := proto.NewWriter(proto.GXMobEnter)
		spawn.U32(mobID)
		spawn.Bool(false)
		spawn.U16(template)
		spawn.F32(100)
		spawn.F32(2)
		spawn.F32(100)
		a.applyWorldPacket(spawn.Bytes())
		return a
	}
	move := func(a *App, x, y float32, ms uint32) {
		w := proto.NewWriter(proto.GXMobMove)
		w.U32(mobID)
		w.U8(1)
		w.F32(x)
		w.F32(y)
		w.F32(100)
		w.U32(ms)
		a.applyWorldPacket(w.Bytes())
	}
	strike := func(a *App, op proto.Opcode, from, to uint32, hp uint16) {
		w := proto.NewWriter(op)
		w.U8(0)
		w.U32(from)
		w.U32(to)
		w.U16(hp)
		w.U16(0)
		w.U16(0)
		a.applyWorldPacket(w.Bytes())
	}

	t.Run("native running pace", func(t *testing.T) {
		a := newApp(902)
		m := a.wv.mobs[mobID]
		legacyMove := func(x float32) {
			w := proto.NewWriter(proto.MobMove)
			w.U32(mobID)
			w.U8(1)
			w.F32(x)
			w.F32(100)
			a.applyWorldPacket(w.Bytes())
		}
		speed := func(tier uint8) {
			w := proto.NewWriter(proto.MobSpeed)
			w.U32(mobID)
			w.U8(3)
			w.U8(tier)
			a.applyWorldPacket(w.Bytes())
		}
		sample := func(elapsed, want float64) {
			t.Helper()
			a.clock += elapsed
			a.updateMobPosition(m)
			if math.Abs(float64(m.pos[0])-want) > .00002 {
				t.Fatalf("x = %g, want %.5f", m.pos[0], want)
			}
		}
		legacyMove(100.16)
		sample(1.0/60, 100.10611)
		sample(1.0/60, 100.16)
		legacyMove(112)
		sample(.1, 100.79667)
		speed(0)
		sample(0, 100.79667)
		sample(.1, 101.21111)
		speed(255)
		sample(.1, 101.84778)

		move(a, 102, 2, 26)
		speed(0)
		sample(.026, 102)
	})

	t.Run("movement and mob swing", func(t *testing.T) {
		a := newApp(2009)
		a.wv.pos = render.V3{101, 2, 100}
		m := a.wv.mobs[mobID]
		m.yaw, m.heading = 0, 0
		move(a, 104.8, 2, 753)
		a.clock += .05
		a.prepareMonsters()
		if m.pos[0] <= 100 || m.pos[0] >= 101 || m.yaw <= 0 || m.yaw >= math.Pi/2 {
			t.Fatalf("movement or turn snapped at first display: %v yaw=%g", m.pos, m.yaw)
		}
		a.state.Target, a.wv.approachTarget, a.wv.engagedID = mobID, mobID, mobID
		a.wv.travel = true
		if !a.updateActionApproach(&platform.Input{}) || a.wv.travel {
			t.Fatalf("melee approach chased destination %.2f with model in reach at %.2f", m.entity.X, m.pos[0])
		}
		a.clock += .753
		a.prepareMonsters()
		strike(a, proto.MobAttackCharacter, mobID, playerID, 123)
		if !m.actor.Attacking() || m.attackImpact == nil || len(a.combatNumbers) != 0 {
			t.Fatal("first swing or delayed contact was lost")
		}
		started := a.clock
		contact := m.attackImpact.contact
		for frame := 0; frame < 180 && m.attackImpact != nil; frame++ {
			a.clock += a.frameDT
			a.prepareMonsters()
		}
		if m.actor.Time < contact || len(a.combatNumbers) != 1 || a.combatNumbers[0].Amount != 123 {
			t.Fatalf("mob swing result at cursor %g, want one at %g: %+v", m.actor.Time, contact, a.combatNumbers)
		}
		t.Logf("Mob contact at %.3fs including blend, attack cursor %.3fs", a.clock-started, m.actor.Time)

		a = newApp(2009)
		strike(a, proto.MobAttackCharacter, mobID, playerID, 123)
		m = a.wv.mobs[mobID]
		if m.actor == nil || !m.actor.Attacking() || len(a.combatNumbers) != 0 {
			t.Fatal("enter followed by attack in the same packet batch lost the swing")
		}
	})

	for _, tc := range []struct {
		name                       string
		weapon, race, job, variant uint8
		twoHits                    bool
	}{
		{"sword", 1, 1, 2, 0, false},
		{"dual swords", 5, 0, 0, 0, true},
		{"daggers", 9, 2, 2, 0, true},
		{"claws", 15, 2, 2, 1, true},
		{"bow", 13, 1, 2, 0, false},
	} {
		t.Run(tc.name+" lethal contact", func(t *testing.T) {
			weapon := tc.weapon
			a := newApp(2009)
			a.state.Target = mobID
			a.wv.pos = render.V3{98, 2, 100}
			a.state.Inventory[[2]uint8{0, 5}] = proto.InventoryItem{Type: weapon, TypeID: 1}
			actor, err := scene.BuildCharacter(assets, scene.Appearance{Race: tc.race, Job: tc.job, Height: 2, Helmet: -1, WeaponType: weapon}, "000_normal")
			if err != nil {
				t.Fatal(err)
			}
			a.wv.self = actor
			a.attackVariant = tc.variant
			move(a, 104, 6, 1000)
			a.prepareMonsters()
			m := a.wv.mobs[mobID]
			strike(a, proto.CharacterMobAutoAttack, playerID, mobID, 100)
			first, second := actor.ContactTimes()
			if tc.twoHits && (first <= 0 || second <= first) {
				t.Fatal("fixture needs two contact markers", first, second)
			}
			if m.incomingHits == 0 || len(a.combatNumbers) != 0 {
				t.Fatal("lethal hit was not scheduled at contact")
			}
			var laterHit *mobActor
			if tc.name == "sword" {
				laterHit = a.waitForMobHit(mobID, 1, 0)
				if laterHit != m {
					t.Fatal("expected a second hit still in flight")
				}
			}
			move(a, 102, 4, 0)
			death := proto.NewWriter(proto.MobDeath)
			death.U32(mobID)
			death.U8(1)
			death.U32(playerID)
			a.applyWorldPacket(death.Bytes())
			a.prepareMonsters()
			if !m.entity.Dead || a.state.Target != 0 || m.dead || len(a.combatNumbers) != 0 ||
				m.position.duration != 0 || m.pos != (render.V3{102, 4, 100}) {
				t.Fatalf("death failed to freeze at the server point while waiting for contact: %v", m.pos)
			}
			if weapon == 13 && (len(a.wv.physicalProjectiles) != 1 || a.wv.physicalProjectiles[0].released) {
				t.Fatal("expected the death notice before arrow release")
			}
			started := a.clock
			advance := func() {
				a.clock += a.frameDT
				actor.Update(float32(a.frameDT))
				a.updatePendingNumbers()
				a.prepareMonsters()
				a.updatePhysicalProjectiles(float32(a.frameDT))
			}
			for frame := 0; frame < 180 && !m.dead; frame++ {
				advance()
			}
			want := uint32(100)
			if tc.twoHits {
				want = 50
			}
			pending := 0
			if laterHit != nil {
				pending = 1
			}
			if !m.dead || m.incomingHits != pending || len(a.combatNumbers) != 1 || a.combatNumbers[0].Amount != want ||
				m.pos != (render.V3{102, 4, 100}) || m.deathAt != a.combatNumbers[0].Started {
				t.Fatalf("death and impact diverged: dead=%t pending=%d pos=%v numbers=%+v", m.dead, m.incomingHits, m.pos, a.combatNumbers)
			}
			if laterHit != nil {
				deathAt := m.deathAt
				for range 2 {
					advance()
				}
				if m.actor.Time <= 0 || m.deathAt != deathAt || m.incomingHits != 1 {
					t.Fatal("another pending hit delayed or interrupted the death clip")
				}
				a.finishMobHit(laterHit)
				if m.incomingHits != 0 || m.deathAt != deathAt {
					t.Fatal("later contact restarted the death")
				}
			}
			if tc.twoHits {
				deathAt := m.deathAt
				for frame := 0; frame < 180 && len(a.combatNumbers) < 2; frame++ {
					advance()
				}
				if len(a.combatNumbers) != 2 || a.combatNumbers[1].Amount != 50 || m.deathAt != deathAt || a.combatNumbers[1].Started <= deathAt {
					t.Fatal("second contact did not retain its number after death", a.combatNumbers)
				}
			}
			t.Logf("%s contact and death at %.3fs, corpse anchored at %v", tc.name, m.deathAt-started, m.pos)
		})
	}
}
