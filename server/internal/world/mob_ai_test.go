package world

import (
	"fmt"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/server/internal/db"
	"testing"
	"time"
)

func luaWorldFixture(t testing.TB, code uint8) (*mapShard, *mobAIHost) {
	t.Helper()
	const zone uint16 = 62080
	old := db.MustDefs()
	defs := *old
	defs.Mobs = map[uint16]*db.MobDef{}
	for _, id := range []uint16{2470, 2471, 2472, 2473, 2464, 2465, 2466, 2467, 1967} {
		defs.Mobs[id] = &db.MobDef{MobID: id, Level: 10, MaxHP: 1000, AI: 0, Country: 2, ChaseRange: 20, ChaseStep: 2, ChaseTimeMS: 200, Size: 1}
		defs.Mobs[id].Attacks[0] = db.MobAttack{OK: 1, Range: 3, TimeMS: 2000}
	}
	id := uint16(2470)
	if code == 115 {
		id = 2471
	}
	defs.Mobs[id].AI = code
	db.SetDefsForTest(&defs)
	sh := shard(zone)
	sh.entities = MapEntities{Mobs: []MapMob{NewStaticMob(defs.Mobs[id], id, 100, 100, nil, 0)}}
	m := &sh.entities.Mobs[0]
	m.noTargetUntil = time.Time{}
	tick := &MobCombatTick{}
	h := &mobAIHost{mob: m, shard: sh, mapID: zone, now: time.Now(), tick: tick, players: []PlayerCombatSnapshot{{MapID: zone, CharacterID: 99, Alive: true, PosX: 102, PosZ: 100, Cell: cellFor(102, 100), HP: 500}}}
	t.Cleanup(func() {
		if sh.lua != nil {
			sh.lua.Close()
		}
		realm.shardsMu.Lock()
		delete(realm.shards, zone)
		realm.shardsMu.Unlock()
		db.SetDefsForTest(old)
	})
	return sh, h
}
func eventOK(t *testing.T, h *mobAIHost, event string, args ...any) {
	t.Helper()
	if !h.event(event, args...) {
		t.Fatalf("event %s: %v", event, h.mob.lua.fault)
	}
}

func TestLuaWorldPoliciesAcquireAndRecruit(t *testing.T) {
	for _, code := range []uint8{0, 1, 2, 3, 4, 5, 52, 104} {
		t.Run(fmt.Sprint(code), func(t *testing.T) {
			sh, h := luaWorldFixture(t, code)
			// A helper with the same template, an aggressive different template, and
			// a reactive different template distinguish the three original help rules.
			caller := sh.entities.Mobs[0]
			for i, ai := range []uint8{0, 1, 0} {
				m := NewStaticMob(db.MustDefs().Mobs[2472], 2472, 103+float32(i), 100, nil, 0)
				m.AI = ai
				if i == 0 {
					m.MobID = caller.MobID
				}
				sh.entities.Mobs = append(sh.entities.Mobs, m)
			}
			h.mob = &sh.entities.Mobs[0]
			eventOK(t, h, "idle")
			acquires := code == 1 || code == 3 || code == 104
			if (h.mob.AggroTarget == 99) != acquires {
				t.Fatalf("AI%d acquisition target=%d", code, h.mob.AggroTarget)
			}
			h.mob.AggroTarget = 99
			h.notifyTarget()
			wants := []bool{code == 2 || code == 5, code == 3 || code == 5, code == 5}
			for i, want := range wants {
				if (sh.entities.Mobs[i+1].AggroTarget == 99) != want {
					t.Fatalf("AI%d helper%d target=%d", code, i, sh.entities.Mobs[i+1].AggroTarget)
				}
			}
			if code == 104 {
				eventOK(t, h, "combat", 100, 0)
				if len(h.tick.Attacks) != 1 {
					t.Fatal("missing Lua source lost native attack")
				}
			}
		})
	}
}
func TestLuaWorldMachineChangesActualCooldownAndStats(t *testing.T) {
	_, h := luaWorldFixture(t, 10)
	for range 3 {
		eventOK(t, h, "idle")
	}
	h.mob.CurrentHP = 400
	eventOK(t, h, "idle")
	if h.mob.lua.speed != 2 {
		t.Fatalf("threshold did not empower mob: %+v", h.mob.lua)
	}
	h.mob.slotReadyAt = [3]time.Time{}
	h.mob.nextAttackAt = time.Time{}
	h.mob.useSlot(0, h.now)
	if got := h.mob.nextAttackAt.Sub(h.now); got != time.Second {
		t.Fatalf("speed2 cooldown=%v", got)
	}
	for name, v := range map[string]float64{"set_damage_multiplier": 2.5, "set_physical_defense_multiplier": 2, "set_attack_speed_multiplier": .5} {
		if _, err := h.Call(name, []any{v}); err != nil {
			t.Fatal(err)
		}
	}
	h.mob.def.Defense = 70
	h.mob.def.ShieldDefense = 10
	combat := mobCombatOf(h.mob)
	if combat.DamageFactor != 2.5 || combat.Guard[0] != 140 || combat.Guard[1] != 150 {
		t.Fatalf("AI factors did not reach combat: %+v", combat)
	}
	h.mob.useSlot(0, h.now)
	if got := h.mob.nextAttackAt.Sub(h.now); got != 4*time.Second {
		t.Fatalf("speed0.5 cooldown=%v", got)
	}
}
func TestLuaWorldControllerInvulnerabilityAndMovement(t *testing.T) {
	sh, h := luaWorldFixture(t, 115)
	result, ok := DamageMobFrom(h.mapID, h.mob.ID, 9999, 99)
	if !ok || result.AppliedHP != 0 || result.Killed || sh.entities.Mobs[0].CurrentHP != 1000 {
		t.Fatal("controller was damaged before first tick")
	}
	h.mob.AggroTarget = 99
	h.notifyTarget()
	h.mob.nextAttackAt = h.now.Add(time.Hour)
	eventOK(t, h, "combat", 100, 1)
	if !h.mob.lua.fixed {
		t.Fatal("controller did not enter fixed movement")
	}
	runMobAICommands(h.tick)
	if len(sh.entities.Mobs) != 3 {
		t.Fatalf("summoned %d mobs", len(sh.entities.Mobs)-1)
	}
	// Slice may have moved when the deferred summons were applied.
	h.mob = &sh.entities.Mobs[0]
	h.mob.PosX, h.mob.PosZ = 173, 179
	h.mob.nextChaseAt = time.Time{}
	eventOK(t, h, "idle")
	if !h.mob.lua.arrived {
		t.Fatal("fixed movement did not finish")
	}
	h.mob.lua.fixed = false
	eventOK(t, h, "move_end")
	runMobAICommands(h.tick)
	if sh.entities.Mobs[0].CurrentHP != 0 || sh.entities.Mobs[0].corpseGone {
		t.Fatal("controller was not removed on arrival")
	}
}
func TestLuaWorldSplitUsesDeferredSpawnAndRemoval(t *testing.T) {
	sh, h := luaWorldFixture(t, 116)
	h.mob.CurrentHP = 500
	h.mob.AggroTarget = 99
	h.mob.nextAttackAt = h.now.Add(time.Hour)
	eventOK(t, h, "combat", 50, 1)
	if len(sh.entities.Mobs) != 1 {
		t.Fatal("spawned before deadline")
	}
	h.now = h.now.Add(time.Second)
	eventOK(t, h, "combat", 50, 1)
	if len(sh.entities.Mobs) != 1 {
		t.Fatal("mutated mob slice from inside Lua")
	}
	runMobAICommands(h.tick)
	counts := map[uint16]int{}
	for i := range sh.entities.Mobs {
		m := &sh.entities.Mobs[i]
		if m.CurrentHP > 0 {
			counts[m.MobID]++
			if DistanceXZ(m.PosX, m.PosZ, 100, 100) > 15 {
				t.Fatal("spawn did not use caller coordinates")
			}
		}
	}
	if counts[2470] != 0 || counts[2472] != 1 || counts[2473] != 2 {
		t.Fatalf("split result=%v", counts)
	}
	if sh.entities.Mobs[0].corpseGone || sh.entities.Mobs[0].diedAt.IsZero() {
		t.Fatal("removal skipped native death lifetime")
	}
}
func TestLuaWorldPortalAndStaleGeneration(t *testing.T) {
	sh, h := luaWorldFixture(t, 111)
	sh.entities.Portals = []MapPortal{{PortalID: 1001, FactionOrPortalID: 5}, {PortalID: 1002, FactionOrPortalID: 5}}
	eventOK(t, h, "death", 0)
	runMobAICommands(h.tick)
	if sh.entities.Portals[0].FactionOrPortalID != 0 || sh.entities.Portals[1].FactionOrPortalID != 5 {
		t.Fatal("death opened wrong portals")
	}
	sh.entities.Portals[0].FactionOrPortalID = 5
	eventOK(t, h, "death", 0)
	h.mob.resetAI()
	runMobAICommands(h.tick)
	if sh.entities.Portals[0].FactionOrPortalID != 5 {
		t.Fatal("old generation changed portal after respawn")
	}
}
func TestLuaWorldSummonAttackMaskAndSpeechPackets(t *testing.T) {
	sh, h := luaWorldFixture(t, 0)
	eventOK(t, h, "idle")
	if _, err := h.Call("summon_legacy", []any{int64(1967), float64(10), int64(1), int64(0)}); err != nil {
		t.Fatal(err)
	}
	runMobAICommands(h.tick)
	m := &sh.entities.Mobs[1]
	mh := &mobAIHost{mob: m, shard: sh, mapID: h.mapID, now: h.now, tick: h.tick}
	if !m.ensureAI(mh) || m.lua.mask != 1 {
		t.Fatal("summon lost original attack mask during initialization")
	}
	for _, indexed := range []bool{false, true} {
		frame := buildMobAIMessage(123, 6401, "Hola", indexed)
		speech, err := proto.DecodeMobSpeech(frame[2:])
		if err != nil || speech.MobID != 123 {
			t.Fatalf("native speech: %+v %v", speech, err)
		}
		if indexed && speech.MessageID != 6401 || !indexed && speech.Text != "Hola" {
			t.Fatalf("wrong speech payload: %+v", speech)
		}
	}
}
func BenchmarkLuaWorldIdle(b *testing.B) {
	_, h := luaWorldFixture(b, 0)
	if !h.mob.ensureAI(h) {
		b.Fatal(h.mob.lua.fault)
	}
	b.ResetTimer()
	b.ReportAllocs()
	for range b.N {
		if !h.event("idle") {
			b.Fatal(h.mob.lua.fault)
		}
	}
}

func TestLuaWorldCatalogRunsWithRealPrimitives(t *testing.T) {
	codes := []uint8{104, 111, 112, 113, 114, 115, 116, 117, 118}
	for code := uint8(0); code <= 36; code++ {
		codes = append(codes, code)
	}
	for _, code := range codes {
		t.Run(fmt.Sprint(code), func(t *testing.T) {
			_, h := luaWorldFixture(t, code)
			db.MustDefs().Mobs[1703] = &db.MobDef{MobID: 1703, MaxHP: 1000, Level: 10, Country: 2}
			h.mob.aggro = map[uint32]int64{99: 100}
			h.mob.AggroTarget = 99
			eventOK(t, h, "attacked", 99)
			h.notifyTarget()
			for i := range 100 {
				h.now = h.now.Add(5 * time.Second)
				h.mob.CurrentHP = uint32(1000 - i*9)
				h.mob.nextAttackAt = h.now.Add(time.Second)
				eventOK(t, h, "idle")
				eventOK(t, h, "combat", int(h.mob.CurrentHP/10), i)
			}
			eventOK(t, h, "death", 100)
			eventOK(t, h, "reset")
		})
	}
}
func TestLuaWorldRecoveryHonorsStatusAndDeadline(t *testing.T) {
	_, h := luaWorldFixture(t, 0)
	h.mob.CurrentHP = 500
	h.mob.def.RecoverValue = 100
	h.mob.returningHome = true
	eventOK(t, h, "idle")
	if h.mob.CurrentHP != 500 {
		t.Fatal("return-home recovery bypassed native idle guard")
	}
	h.mob.returningHome = false
	eventOK(t, h, "idle")
	if h.mob.CurrentHP != 600 {
		t.Fatal("idle recovery did not apply")
	}
	eventOK(t, h, "idle")
	if h.mob.CurrentHP != 600 {
		t.Fatal("recovery ignored five-second deadline")
	}
	h.now = h.now.Add(5 * time.Second)
	eventOK(t, h, "idle")
	if h.mob.CurrentHP != 700 {
		t.Fatal("recovery did not resume at deadline")
	}
}
