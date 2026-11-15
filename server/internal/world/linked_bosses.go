package world

import (
	"math/rand/v2"
	"time"
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
	if b.random {
		b.next = now.Add(bossRandomDelay(b))
		return
	}
	if !spawnEventActor(s, b.principal, b.mobs[0], b.pos, now) {
		b.next = now.Add(time.Second)
		return
	}
	for _, g := range b.guards {
		populateEventSpawn(s, g, 0, now)
	}
}
func bossRandomDelay(b *linkedBoss) time.Duration {
	delay := b.randomMin
	if seconds := int64(b.randomRange / time.Second); seconds > 0 {
		delay += time.Duration(rand.Int64N(seconds)) * time.Second
	}
	return delay
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
		if b.random {
			delay = bossRandomDelay(b)
		}
		b.next = now.Add(mobCorpseDelay + max(delay, 0))
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
			pos = b.randomPositions[rand.IntN(3)]
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
