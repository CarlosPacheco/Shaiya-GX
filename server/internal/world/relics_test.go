package world

import (
	"testing"
	"time"

	"github.com/Spelunkern/Shaiya-GX/server/internal/db"
)

func relicFixture(t *testing.T) *relic {
	t.Helper()
	oldDefs := db.MustDefs()
	fixtureDefs := *oldDefs
	fixtureDefs.Mobs = make(map[uint16]*db.MobDef)
	for _, id := range []uint16{52, 252, 152, 77, 69, 73, 1974, 1977, 65, 66, 68} {
		fixtureDefs.Mobs[id] = &db.MobDef{MobID: id, MaxHP: 100, RespawnSeconds: 10, Country: 2}
	}
	db.SetDefsForTest(&fixtureDefs)
	t.Cleanup(func() { db.SetDefsForTest(oldDefs) })
	withTiers(t)
	oldLoaded, oldConfig, oldMaps, oldBosses, oldPortals, oldScore := relics.loaded, relics.config, relics.byMap, relics.bosses, relics.portals, relics.score
	oldReset := relics.resetPending
	relics.loaded = true
	relics.config = relicConfig{}
	relics.byMap = map[uint16][]*relic{}
	relics.bosses = map[uint16][]*linkedBoss{}
	relics.portals = nil
	relics.score = [2]uint32{1, 1}
	relics.resetPending = false
	const mapID = 61071
	RegisterMapGrid(mapID, 2048, make([]byte, 2048*2048/8))
	s := shard(mapID)
	r := &relic{mapID: mapID, key: uint32(mapID)<<16 | 1, id: 0, pos: [3]float32{100, 40, 100}, mobs: [3]uint16{52, 252, 152}, guards: []*eventSpawn{{pos: [3]float32{105, 40, 100}, radius: 1, count: 1, mobs: []uint16{77, 69, 73}}}}
	r.principal = allocateEventActor(s, mapID, r, nil, nil, true)
	g := r.guards[0]
	g.actors = append(g.actors, allocateEventActor(s, mapID, r, nil, g, false))
	spawnRelicLocked(s, r, time.Now())
	relics.byMap[mapID] = []*relic{r}
	if err := db.SaveKeeps([]db.Keep{relicKeep(r)}); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		p, _ := db.Pool(db.GameData)
		p.Exec(`DELETE FROM PS_GameData_Keeps WHERE KeepID=?`, r.key)
		relics.loaded, relics.config, relics.byMap, relics.bosses, relics.portals, relics.score = oldLoaded, oldConfig, oldMaps, oldBosses, oldPortals, oldScore
		relics.resetPending = oldReset
		realm.shardsMu.Lock()
		delete(realm.shards, mapID)
		realm.shardsMu.Unlock()
		realm.gridsMu.Lock()
		delete(realm.grids, mapID)
		realm.gridsMu.Unlock()
	})
	return r
}
func killRelicFixture(t *testing.T, r *relic, country uint8) MapMob {
	t.Helper()
	m := shard(r.mapID).entities.Mobs[r.principal.slot]
	result, ok := DamageMob(r.mapID, m.ID, m.MaxHP)
	if !ok || !result.Killed {
		t.Fatal("fixture relic did not die")
	}
	eventMobDestroyed(&ConnectedPlayer{CharacterID: 990071, faction: country}, r.mapID, &result.Mob, nopLog{})
	return result.Mob
}
func TestRelicCapturePersistenceRespawnAndReset(t *testing.T) {
	r := relicFixture(t)
	m := killRelicFixture(t, r, 0)
	s := shard(r.mapID)
	if r.owner != 1 || r.last != 1 || r.standing || s.entities.Mobs[r.guards[0].actors[0].slot].CurrentHP != 0 {
		t.Fatal("capture did not remove dependents/change ownership")
	}
	rows, err := db.ReadKeeps()
	if err != nil || rows[r.key].Country != 0x10 {
		t.Fatalf("packed persistence=%+v error=%v", rows[r.key], err)
	}
	if kcWorld[0].count != 1 {
		t.Fatal("capture score")
	}
	eventMobDestroyed(&ConnectedPlayer{faction: 0}, r.mapID, &m, nopLog{})
	if kcWorld[0].count != 1 {
		t.Fatal("duplicate capture scored")
	}
	TickRelics(r.next.Add(-time.Nanosecond), nopLog{})
	if r.standing {
		t.Fatal("early principal respawn")
	}
	TickRelics(r.next, nopLog{})
	if !r.standing || s.entities.Mobs[r.principal.slot].MobID != 252 || s.entities.Mobs[r.guards[0].actors[0].slot].MobID != 69 {
		t.Fatal("owner variants not restored")
	}
	// Reset preserves the last conqueror when the base is neutral; portal links
	// use that last conqueror even while the visible owner returns to neutral.
	now := time.Now()
	ResetRelics(now, nopLog{})
	if r.owner != 0 || r.last != 1 || r.standing || !r.next.Equal(now.Add(10*time.Minute)) {
		t.Fatal("reset ownership/deadline")
	}
	if kcWorld[0].count != 1 {
		t.Fatal("reset incorrectly scored a capture")
	}
	TickRelics(now.Add(10*time.Minute), nopLog{})
	if !r.standing || s.entities.Mobs[r.principal.slot].MobID != 52 {
		t.Fatal("neutral base did not respawn")
	}
}
func TestRelicSaveFailureRetriesBeforePublishingAndScoring(t *testing.T) {
	r := relicFixture(t)
	p, _ := db.Pool(db.GameData)
	if _, err := p.Exec(`CREATE TEMP TRIGGER reject_relic_save BEFORE UPDATE ON PS_GameData_Keeps BEGIN SELECT RAISE(ABORT,'isolated failure'); END`); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { p.Exec(`DROP TRIGGER IF EXISTS reject_relic_save`) })
	killRelicFixture(t, r, 0)
	if r.owner != 0 || r.pending == nil || kcWorld[0].count != 0 {
		t.Fatal("failed save published capture")
	}
	p.Exec(`DROP TRIGGER reject_relic_save`)
	TickRelics(time.Now(), nopLog{})
	if r.owner != 1 || r.pending != nil || kcWorld[0].count != 1 {
		t.Fatal("capture retry failed")
	}
}

func TestRelicLinkedPortalsPersistHysteresisAndFactionAccess(t *testing.T) {
	r := relicFixture(t)
	s := shard(r.mapID)
	s.entities.Portals = []MapPortal{{Index: 0, PortalID: 1060, FactionOrPortalID: 3}, {Index: 1, PortalID: 1061, FactionOrPortalID: 3}}
	inputs := []*relicPortal{{key: 0x10000071, mapID: r.mapID, portalID: 1060, refs: [2][]uint32{{r.key}, nil}}, {key: 0x10000072, mapID: r.mapID, portalID: 1061, refs: [2][]uint32{nil, {r.key}}}}
	next := uint32(1)
	var defaults []db.Keep
	if err := bindRelicPortals(inputs, nil, &next, &defaults); err != nil {
		t.Fatal(err)
	}
	if err := db.SaveKeeps(defaults); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		p, _ := db.Pool(db.GameData)
		p.Exec(`DELETE FROM PS_GameData_Keeps WHERE KeepID IN (?,?)`, inputs[0].key, inputs[1].key)
	})
	killRelicFixture(t, r, 0)
	TickRelics(time.Now(), nopLog{})
	light, _ := FindPortalByIndex(r.mapID, 0)
	fury, _ := FindPortalByIndex(r.mapID, 1)
	if !light.IsOpen() || !light.AllowsFaction(0) || light.AllowsFaction(1) || fury.IsOpen() {
		t.Fatal("captured portal access")
	}
	ResetRelics(time.Now(), nopLog{})
	TickRelics(time.Now(), nopLog{})
	light, _ = FindPortalByIndex(r.mapID, 0)
	if !light.AllowsFaction(0) || !light.IsOpen() {
		t.Fatal("neutral reset lost last-conqueror portal state")
	}
	TickRelics(r.next, nopLog{})
	killRelicFixture(t, r, 1)
	TickRelics(time.Now(), nopLog{})
	light, _ = FindPortalByIndex(r.mapID, 0)
	fury, _ = FindPortalByIndex(r.mapID, 1)
	if light.IsOpen() || !fury.IsOpen() || !fury.AllowsFaction(1) || fury.AllowsFaction(0) {
		t.Fatal("portal exclusion/faction access")
	}
	rows, _ := db.ReadKeeps()
	if rows[inputs[0].key].Country != 3 || rows[inputs[1].key].Country != 2 {
		t.Fatal("portal persistence", rows)
	}
}
func TestLinkedBossPhasesFinalGuardsAndPortal(t *testing.T) {
	r := relicFixture(t)
	s := shard(r.mapID)
	s.entities.Portals = []MapPortal{{PortalID: 1001, FactionOrPortalID: 3}}
	b := &linkedBoss{mapID: r.mapID, key: 1, pos: [3]float32{200, 40, 200}, mobs: []uint16{1974, 1977}, portalMap: r.mapID, portalID: 1001, delay: 24 * time.Hour, guards: []*eventSpawn{{pos: [3]float32{205, 40, 200}, radius: 1, count: 1, mobs: []uint16{65, 66, 68}}}}
	initializeBossLocked(r.mapID, b, time.Now())
	for phase, want := range []uint16{1974, 1977} {
		m := s.entities.Mobs[b.principal.slot]
		if m.MobID != want || m.CurrentHP == 0 {
			t.Fatalf("phase %d mob=%d", phase, m.MobID)
		}
		result, ok := DamageMob(r.mapID, m.ID, m.MaxHP)
		if !ok || !result.Killed {
			t.Fatal("boss death")
		}
		eventMobDestroyed(&ConnectedPlayer{faction: 0}, r.mapID, &result.Mob, nopLog{})
		if phase == 0 {
			if b.stage != 1 || b.portalOpen {
				t.Fatal("intermediate phase opened portal")
			}
			TickRelics(b.next, nopLog{})
		}
	}
	if b.stage != 0 || !b.portalOpen || s.entities.Mobs[b.guards[0].actors[0].slot].MobID != 68 {
		t.Fatal("final phase did not spawn configured final guardians")
	}
	p, _ := FindPortalByIndex(r.mapID, 0)
	if !p.IsOpen() {
		t.Fatal("final boss did not open its actual portal")
	}
	if kcWorld[0].count != 0 {
		t.Fatal("boss kill incorrectly scored as relic")
	}
	TickRelics(b.next, nopLog{})
	p, _ = FindPortalByIndex(r.mapID, 0)
	if p.IsOpen() || s.entities.Mobs[b.principal.slot].MobID != 1974 || s.entities.Mobs[b.guards[0].actors[0].slot].MobID != 65 {
		t.Fatal("boss cycle did not close portal/restart phase zero")
	}
}

func TestRelicResetUsesNonneutralBaseAndRejectsStaleDeaths(t *testing.T) {
	r := relicFixture(t)
	r.initial = 2
	stale := killRelicFixture(t, r, 0)
	TickRelics(r.next, nopLog{})
	count := kcWorld[0].count
	eventMobDestroyed(&ConnectedPlayer{faction: 0}, r.mapID, &stale, nopLog{})
	if !r.standing || kcWorld[0].count != count {
		t.Fatal("previous generation death affected replacement")
	}
	ResetRelics(time.Now(), nopLog{})
	if r.owner != 2 || r.last != 2 {
		t.Fatal("reset forced an authored Fury base neutral")
	}
	rows, err := db.ReadKeeps()
	if err != nil {
		t.Fatal(err)
	}
	restored := &relic{key: r.key, initial: 2}
	if err := restoreRelic(restored, rows[r.key]); err != nil {
		t.Fatal(err)
	}
	if restored.owner != 2 || restored.last != 2 {
		t.Fatal("restart lost packed owner/conqueror")
	}
}
func TestLinkedRandomBossRollsChanceAndPosition(t *testing.T) {
	relicFixture(t)
	// A common (non-instance) map id: the fixture's high id is an instance zone.
	const mapID = 913
	RegisterMapGrid(mapID, 2048, make([]byte, 2048*2048/8))
	r := struct{ mapID uint16 }{mapID}
	now := time.Now()
	positions := [][3]float32{{200, 40, 200}, {300, 50, 300}, {400, 60, 400}}
	b := &linkedBoss{mapID: r.mapID, key: 24, mobs: []uint16{1974}, random: true, positions: positions, lastPos: -1}
	initializeBossLocked(r.mapID, b, now)
	s := shard(r.mapID)
	m := s.entities.Mobs[b.principal.slot]
	found := false
	for _, p := range positions {
		if m.PosX == p[0] && m.PosY == p[1] && m.PosZ == p[2] {
			found = true
		}
	}
	if !found || m.CurrentHP == 0 {
		t.Fatal("a random boss without a chance must appear at once at an authored position")
	}
	// A failed roll waits for the mob's own respawn (at least a minute).
	never := &linkedBoss{mapID: r.mapID, key: 25, mobs: []uint16{1974}, random: true, chance: 1e-9, positions: positions, lastPos: -1}
	initializeBossLocked(r.mapID, never, now)
	if s.entities.Mobs[never.principal.slot].CurrentHP != 0 || never.next.Before(now.Add(time.Minute)) {
		t.Fatal("failed roll spawned or retried too soon", never.next.Sub(now))
	}
}

func TestRandomBossRollAvoidsPreviousPosition(t *testing.T) {
	b := &linkedBoss{positions: [][3]float32{{1}, {2}, {3}}, lastPos: -1}
	for range 200 {
		before := b.lastPos
		if _, ok := bossRoll(b); !ok || b.lastPos == before {
			t.Fatal("repeated the previous position")
		}
	}
}

func TestInstanceRandomBossRollsOnceAndNeverReturns(t *testing.T) {
	b := &linkedBoss{mapID: instanceZoneBase + 7, mobs: []uint16{1}, random: true, chance: 1e-9, positions: [][3]float32{{1}}}
	if _, ok := bossRoll(b); ok {
		t.Fatal("near-zero chance succeeded")
	}
	if !bossRetry(b, time.Now()).IsZero() {
		t.Fatal("an instance retried its random boss")
	}
}

func TestRelicFairDamageCreditAndOfflineCountryFallback(t *testing.T) {
	r := relicFixture(t)
	s := shard(r.mapID)
	id := s.entities.Mobs[r.principal.slot].ID
	first, ok := DamageMobFrom(r.mapID, id, 30, 10)
	if !ok || first.Killed {
		t.Fatal("first contribution")
	}
	second, ok := DamageMobFrom(r.mapID, id, 10, 20)
	if !ok || second.Killed {
		t.Fatal("second contribution")
	}
	final, ok := DamageMobFrom(r.mapID, id, 60, 20)
	if !ok || !final.Killed || eventDamageWinner(&final.Mob) != 20 {
		t.Fatal("damage aggregation/final-hit accounting")
	}
	if eventDamageWinner(&first.Mob) != 10 {
		t.Fatal("later damage mutated an earlier snapshot")
	}
	tied := MapMob{eventDamage: []eventDamageEntry{{ID: 10, Amount: 50}, {ID: 20, Amount: 50}}}
	if eventDamageWinner(&tied) != 10 {
		t.Fatal("tie lost first attacker")
	}
	eventMobDestroyed(&ConnectedPlayer{CharacterID: 20, faction: 0}, r.mapID, &final.Mob, nopLog{})
	TickRelics(r.next, nopLog{})
	final, ok = DamageMobFrom(r.mapID, id, 100, 20)
	if !ok || !final.Killed {
		t.Fatal("second cycle")
	}
	if eventCreditedPlayer(nil, r.mapID, &final.Mob) != nil {
		t.Fatal("offline caster unexpectedly found")
	}
	eventMobDestroyed(nil, r.mapID, &final.Mob, nopLog{})
	if r.owner != 2 {
		t.Fatal("native no-credited-user path must flip the prior Light owner to Fury")
	}
}

func TestRelicMobileCompanionsUseOrdinaryRoamAndParentLifetime(t *testing.T) {
	r := relicFixture(t)
	sh := shard(r.mapID)
	def := db.MustDefs().Mobs[77]
	def.NormalStep, def.ChaseStep, def.NormalTimeMS = 2, 4, 1000
	g := r.guards[0]
	a := g.actors[0]
	if !spawnEventActor(sh, a, 77, [3]float32{105, 40, 100}, time.Now()) {
		t.Fatal("companion did not spawn")
	}
	m := &sh.entities.Mobs[a.slot]
	if m.Area == nil || m.event != a || m.stationary() {
		t.Fatal("ordinary mobile companion not configured")
	}
	m.nextMoveAt = time.Time{}
	moves := TickMobCombat([]PlayerCombatSnapshot{{MapID: r.mapID, PosX: 105, PosZ: 100}}, 100).Moves
	if len(moves) != 1 || m.PosX == 105 && m.PosZ == 100 {
		t.Fatal("linked mob did not wander normally")
	}
	killRelicFixture(t, r, 0)
	if sh.entities.Mobs[a.slot].CurrentHP != 0 {
		t.Fatal("wandering companion outlived parent")
	}
}
