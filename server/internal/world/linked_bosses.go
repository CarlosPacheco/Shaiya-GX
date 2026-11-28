package world

import (
	"math/rand/v2"
	"time"

	"github.com/Spelunkern/Shaiya-GX/server/internal/db"
)

func initializeBossLocked(mapID uint16, b *linkedBoss, now time.Time) {
	s := shard(mapID)
	s.mu.Lock()
	defer s.mu.Unlock()
	b.mapID = mapID
	b.principal = allocateEventActor(s, mapID, nil, b, nil, true)
	for _, g := range b.guards {
		for range g.count {
			g.actors = append(g.actors, allocateEventActor(s, mapID, nil, b, g, false))
		}
	}
	relics.bosses[mapID] = append(relics.bosses[mapID], b)
	pos := b.pos
	if b.random {
		var ok bool
		if pos, ok = bossRoll(b); !ok {
			b.next = bossRetry(b, now)
			return
		}
	}
	if !spawnEventActor(s, b.principal, b.mobs[0], pos, now) {
		b.next = now.Add(time.Second)
		return
	}
	for _, g := range b.guards {
		populateEventSpawn(s, g, 0, now)
	}
}

// bossRoll is the [RandomRespawn] appearance roll: chance percent (0 means
// always) and a random authored position, not the previous one when there
// are several. The chosen position is kept for the boss's later phases.
func bossRoll(b *linkedBoss) ([3]float32, bool) {
	if b.chance > 0 && rand.Float64()*100 >= b.chance {
		return [3]float32{}, false
	}
	i := rand.IntN(len(b.positions))
	if len(b.positions) > 1 && i == b.lastPos {
		i = (i + 1 + rand.IntN(len(b.positions)-1)) % len(b.positions)
	}
	b.lastPos, b.spawnPos = i, b.positions[i]
	return b.spawnPos, true
}

// bossRetry schedules the next roll after a failed one: the first phase's
// own respawn (Attackspecial3), at least a minute. An instance rolls once,
// at creation, so a failed roll leaves it without the boss.
func bossRetry(b *linkedBoss, now time.Time) time.Time {
	if isInstanceZone(b.mapID) {
		return time.Time{}
	}
	delay := time.Minute
	if def := db.MustDefs().Mobs[b.mobs[0]]; def != nil {
		delay = max(delay, time.Duration(def.RespawnSeconds)*time.Second)
	}
	return now.Add(delay)
}
func bossPortalMap(b *linkedBoss) uint16 {
	if realMap(b.mapID) == b.portalMap {
		return b.mapID
	}
	return b.portalMap
}
func bossDestroyedLocked(s *mapShard, b *linkedBoss, now time.Time, out *[]eventPublication) {
	m := &s.entities.Mobs[b.principal.slot]
	for _, g := range b.guards {
		for _, a := range g.actors {
			removeEventActor(s, a)
		}
	}
	b.stage++
	if b.stage >= len(b.mobs) {
		b.stage = 0
		// No portal shard lock while holding the source shard. The actual write
		// follows below in tickBossLocked, before the replacement can appear.
		delay := b.delay + time.Duration(rand.IntN(120)-60)*time.Minute
		b.next = now.Add(mobCorpseDelay + max(delay, 0))
		if b.random && isInstanceZone(b.mapID) {
			b.next = time.Time{} // an instance's random boss does not come back
		}
		b.portalOpen = true
		for _, g := range b.guards {
			populateEventSpawn(s, g, len(b.mobs), now)
		}
	} else {
		b.next = now.Add(mobCorpseDelay + time.Duration(m.def.RespawnSeconds)*time.Second)
	}
}
func tickBossLocked(s *mapShard, b *linkedBoss, now time.Time, out *[]eventPublication) {
	m := &s.entities.Mobs[b.principal.slot]
	if m.CurrentHP == 0 && !b.next.IsZero() && !now.Before(b.next) {
		if b.stage == 0 {
			for _, g := range b.guards {
				for _, a := range g.actors {
					removeEventActor(s, a)
				}
			}
			b.portalOpen = false
		}
		pos := b.pos
		if b.random {
			pos = b.spawnPos
			if b.stage == 0 {
				var ok bool
				if pos, ok = bossRoll(b); !ok {
					b.next = bossRetry(b, now)
					return
				}
			}
		}
		if spawnEventActor(s, b.principal, b.mobs[b.stage], pos, now) {
			b.next = time.Time{}
			for _, g := range b.guards {
				populateEventSpawn(s, g, b.stage, now)
			}
		}
	}
	if m.CurrentHP > 0 {
		for _, g := range b.guards {
			tickEventSpawn(s, g, b.stage, now)
		}
	}
}

// Each instance gets its own linked boss phases. Primary-map obelisks are
// deliberately not copied into party/guild instances.
func initializeInstanceBosses(zone, base uint16) {
	relics.Lock()
	defer relics.Unlock()
	if !relics.loaded {
		return
	}
	for _, source := range relics.config.bosses {
		if source.mapID != base {
			continue
		}
		b := *source
		b.stage = 0
		b.principal = nil
		b.guards = nil
		b.portalOpen = false
		for _, g := range source.guards {
			copy := *g
			copy.actors = nil
			copy.next = time.Time{}
			b.guards = append(b.guards, &copy)
		}
		initializeBossLocked(zone, &b, time.Now())
	}
}

func syncBossPortal(b *linkedBoss) {
	if b.portalID <= 1000 {
		return
	}
	country := int32(3)
	if b.portalOpen {
		country = 0
	}
	setEventPortal(bossPortalMap(b), b.portalID, country)
}
