package world

import (
	"testing"
	"time"

	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/server/internal/db"
)

func guardFixture(t *testing.T, mapID uint16) *mapShard {
	t.Helper()
	old := npcDefs[npcTypeGuard-1]
	t.Cleanup(func() { npcDefs[npcTypeGuard-1] = old })
	npcDefs[npcTypeGuard-1] = []npcDef{{Country: 0, MoveDistance: 16, MoveSpeed: 8000}}
	n, _ := NewPatrolNpc(npcTypeGuard, 1, []NpcPatrolPoint{{X: 100, Y: 10, Z: 100}})
	AddEntities(mapID, MapEntities{NPCs: []MapNpc{n}})
	s := shard(mapID)
	t.Cleanup(func() {
		s.mu.Lock()
		s.entities, s.guardClaims = MapEntities{}, nil
		s.mu.Unlock()
	})
	return s
}

func TestGuardChasesStealthAndReturnsWithinNativeLeash(t *testing.T) {
	const id = 65481
	s := guardFixture(t, id)
	now := time.Unix(100, 0)
	target := PlayerCombatSnapshot{CharacterID: 12, MapID: id, PosX: 121, PosY: 10, PosZ: 100, Alive: true, Faction: 1, Invisible: true}
	tick := func() npcTick { return tickNpcs(now, []PlayerCombatSnapshot{target}, 64) }
	if r := tick(); len(r.moves)+len(r.strikes) != 0 {
		t.Fatal("guard detected a target beyond 20")
	}
	target.PosX = 120
	r := tick()
	if len(r.moves) != 1 || len(r.strikes) != 0 {
		t.Fatal("guard failed to start pursuit of a stealthed player")
	}
	move, err := proto.DecodeNpcMove(r.moves[0].Move[2:])
	if err != nil || move.Motion != 1 || move.X != 116 || !s.entities.NPCs[0].guardReady.Equal(now.Add(2*time.Second)) {
		t.Fatalf("guard chase step %+v, %v", move, err)
	}
	target.PosX = 131
	now = now.Add(2 * time.Second)
	r = tick()
	if len(r.moves) != 1 || len(r.strikes) != 0 || s.entities.NPCs[0].PosX != 100 {
		t.Fatal("guard did not return to the route when the enemy crossed the leash")
	}
}

func TestGuardSpareAlliesAndKillEnemyMobInOneStrike(t *testing.T) {
	const id = 65482
	s := guardFixture(t, id)
	now := time.Unix(100, 0)
	s.entities.Mobs = []MapMob{
		{ID: 5, CurrentHP: 10, PosX: 100, PosY: 10, PosZ: 100, def: &db.MobDef{Country: 0}},
		{ID: 6, CurrentHP: 180000, PosX: 102, PosY: 10, PosZ: 100, def: &db.MobDef{Country: 1}, respawnDelay: time.Minute, returningHome: true},
	}
	r := tickNpcs(now, []PlayerCombatSnapshot{{CharacterID: 88, MapID: id, PosX: 101, PosY: 10, PosZ: 100, Alive: true}}, 64)
	if len(r.strikes) != 1 || r.strikes[0].target != (guardTarget{id: 6, mob: true}) || s.entities.Mobs[0].CurrentHP != 10 ||
		s.entities.Mobs[1].CurrentHP != 0 || !s.entities.Mobs[1].respawnAt.Equal(now.Add(time.Minute)) {
		t.Fatal("guard did not preserve its ally and kill the hostile returning mob")
	}
	frame := buildNpcAttack(r.strikes[0].guardID, 6, true, r.strikes[0].hp)
	got, err := proto.DecodeNpcAttack(frame[2:])
	if err != nil || proto.Op(frame[2:]) != proto.MapNpcAttackMob || got.Target != 6 || got.HP != 65535 {
		t.Fatalf("bad guard attack frame %+v, %v", got, err)
	}
}

func TestGuardStrikeIgnoresPlayerImmunityAndRejectsStaleMap(t *testing.T) {
	const id = 65483
	s, _ := gmWireSession(t, db.Character{ID: 1999999990, MapID: id, Level: 20, PosX: 101, PosY: 10, PosZ: 100, CurrentHP: 500}, true)
	s.gmImmune = true
	hit := npcGuardStrike{mapID: id - 1, guardID: 7, x: 100, y: 10, z: 100}
	if s.applyIncomingMobDamage(pendingDamage{Guard: &hit}) || s.char.CurrentHP != 500 {
		t.Fatal("strike survived a map change")
	}
	hit.mapID = id
	if !s.applyIncomingMobDamage(pendingDamage{Guard: &hit}) || s.char.CurrentHP != 0 {
		t.Fatal("GM immunity prevented an explicitly unprotected guard strike")
	}
}

func TestNpcPatrolReversesAndInterpolatesWaypointHeight(t *testing.T) {
	const id = 65484
	guardFixture(t, id)
	now := time.Unix(150, 0)
	n, _ := NewPatrolNpc(npcTypeGuard, 1, []NpcPatrolPoint{{X: 100, Y: 10, Z: 80}, {X: 132, Y: 18, Z: 80}})
	if !n.stepPatrol(now, id) || n.PosX != 116 || n.PosY != 14 || n.Angle != 270 || n.nextMoveAt.Sub(now) != 8*time.Second {
		t.Fatalf("wrong NPC patrol step %+v", n)
	}
	now = n.nextMoveAt
	if !n.stepPatrol(now, id) || n.PosX != 132 || n.PosY != 18 || n.patrolIndex != 0 || !n.backward ||
		n.nextMoveAt.Sub(now) < 18*time.Second || n.nextMoveAt.Sub(now) >= 28*time.Second {
		t.Fatalf("wrong NPC patrol arrival %+v", n)
	}
	if !n.stepPatrol(n.nextMoveAt, id) || n.PosX != 116 || n.PosY != 14 || n.Angle != 90 {
		t.Fatalf("wrong NPC patrol return %+v", n)
	}
}

func TestSummonedGuardWaitsAtSpawnThenAttacks(t *testing.T) {
	const mapID = 65485
	RegisterMapGrid(mapID, 200, nil)
	old := npcDefs[npcTypeGuard-1]
	t.Cleanup(func() { npcDefs[npcTypeGuard-1] = old })
	npcDefs[npcTypeGuard-1] = []npcDef{{Type: npcTypeGuard, TypeID: 1, Country: FactionLight}}
	s, packets := gmWireSession(t, db.Character{ID: 990025, MapID: mapID, PosX: 100, PosY: 10, PosZ: 100}, true)
	if !s.handleGMCommand("/nmake 8 1") {
		t.Fatal("guard command was not handled")
	}
	gmResponse(t, packets)
	sh := shard(mapID)
	t.Cleanup(func() {
		sh.mu.Lock()
		sh.entities, sh.guardClaims = MapEntities{}, nil
		sh.mu.Unlock()
	})
	npcs := EntitiesOnMap(mapID).NPCs
	if len(npcs) != 1 || !npcs[0].Summoned || len(npcs[0].patrol) != 1 {
		t.Fatal("summoned guard needs a single anchor")
	}
	n := npcs[0]
	p := PlayerCombatSnapshot{CharacterID: 88, MapID: mapID, PosX: n.PosX + 2, PosY: n.PosY, PosZ: n.PosZ, Alive: true, Faction: FactionLight}
	now := time.Unix(100, 0)
	if r := tickNpcs(now, []PlayerCombatSnapshot{p}, 64); len(r.moves) != 0 || len(r.strikes) != 0 {
		t.Fatal("guard should stand still around allies")
	}
	p.Faction = FactionDark
	p.PosX += 6
	r := tickNpcs(now, []PlayerCombatSnapshot{p}, 64)
	if len(r.moves) != 1 || len(r.strikes) != 0 {
		t.Fatal("stationary summoned guard did not pursue an enemy")
	}
	r = tickNpcs(sh.entities.NPCs[0].guardReady, []PlayerCombatSnapshot{p}, 64)
	if len(r.strikes) != 1 || r.strikes[0].target.id != p.CharacterID {
		t.Fatal("summoned guard did not attack after reaching its target")
	}
	p.Alive = false
	r = tickNpcs(sh.entities.NPCs[0].guardReady, []PlayerCombatSnapshot{p}, 64)
	if len(r.moves) != 1 || sh.entities.NPCs[0].point() != n.point() {
		t.Fatal("summoned guard did not return to its spawn")
	}
}

// Skill stealth is seen through by guards, but a GM in /char on or /attack on
// mode (the original bVisible/bAttackable flags) is invisible to them.
func TestGuardsIgnoreGMUndetectedPlayers(t *testing.T) {
	const id = 65483
	guardFixture(t, id)
	now := time.Unix(100, 0)
	target := PlayerCombatSnapshot{CharacterID: 12, MapID: id, PosX: 110, PosY: 10, PosZ: 100, Alive: true, Faction: 1, GMUndetected: true}
	if r := tickNpcs(now, []PlayerCombatSnapshot{target}, 64); len(r.moves)+len(r.strikes) != 0 {
		t.Fatal("guard reacted to a GM-undetected player")
	}
	target.GMUndetected = false
	if r := tickNpcs(now, []PlayerCombatSnapshot{target}, 64); len(r.moves)+len(r.strikes) == 0 {
		t.Fatal("guard stopped reacting to an ordinary enemy")
	}
}
