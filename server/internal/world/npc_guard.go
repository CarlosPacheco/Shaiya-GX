package world

import (
	"math"
	"time"

	"github.com/Spelunkern/Shaiya-GX/internal/mapnav"
	"github.com/Spelunkern/Shaiya-GX/server/internal/packet"
)

const (
	npcTypeGuard   = 8
	guardSight     = 20
	guardLeash     = 30
	guardContact   = 3
	deathTypeGuard = 3
)

type guardTarget struct {
	id  uint32
	mob bool
}

type npcGuardStrike struct {
	mapID   uint16
	guardID uint32
	target  guardTarget
	mob     MapMob
	x, y, z float32
	hp      uint32
}

type npcTick struct {
	moves   []entityMove
	strikes []npcGuardStrike
}

func tickNpcs(now time.Time, players []PlayerCombatSnapshot, limit int) npcTick {
	byMap := make(map[uint16][]PlayerCombatSnapshot)
	for _, p := range players {
		byMap[p.MapID] = append(byMap[p.MapID], p)
	}
	var out npcTick
	for mapID, viewers := range byMap {
		s := shard(mapID)
		positions := make([][2]float32, 0, len(viewers))
		for _, p := range viewers {
			positions = append(positions, [2]float32{p.PosX, p.PosZ})
		}
		buckets := bucketViewers(positions)
		s.mu.Lock()
		for target, expires := range s.guardClaims {
			if !now.Before(expires) {
				delete(s.guardClaims, target)
			}
		}
		budget := limit
		for i := range s.entities.NPCs {
			n := &s.entities.NPCs[i]
			if budget == 0 {
				break
			}
			if !buckets.nearAny(n.PosX, n.PosZ, ViewportRadius) {
				continue
			}
			before := len(out.moves) + len(out.strikes)
			if n.Type == npcTypeGuard && len(n.patrol) > 0 && n.updateGuard(now, mapID, s, viewers, &out) {
				budget -= len(out.moves) + len(out.strikes) - before
				continue
			}
			if len(n.patrol) > 1 && !now.Before(n.nextMoveAt) && !npcInUse(n.ID) && n.stepPatrol(now, mapID) {
				out.moves = append(out.moves, n.relocate(mapID, mobMotionWalk))
				budget--
			}
		}
		s.mu.Unlock()
	}
	return out
}

func (n *MapNpc) point() mapnav.Point {
	return mapnav.Point{n.PosX, n.PosY, n.PosZ}
}

func (n *MapNpc) findGuardTarget(s *mapShard, players []PlayerCombatSnapshot, country uint8, home NpcPatrolPoint, locked guardTarget) (guardTarget, mapnav.Point, *MapMob) {
	var target guardTarget
	var point mapnav.Point
	var mob *MapMob
	best := float32(guardSight)
	if locked.id != 0 {
		best = guardLeash
	}
	consider := func(candidate guardTarget, x, y, z float32) bool {
		if locked.id != 0 && locked != candidate || locked.id == 0 && !s.guardClaims[candidate].IsZero() ||
			DistanceXZ(home.X, home.Z, x, z) > guardLeash {
			return false
		}
		d := DistanceXZ(n.PosX, n.PosZ, x, z)
		if d > best || d == best && target.id != 0 && candidate.id >= target.id {
			return false
		}
		target, point, best = candidate, mapnav.Point{x, y, z}, d
		return true
	}
	for _, p := range players {
		if p.Alive && p.Faction != country && !p.GMUndetected {
			consider(guardTarget{id: p.CharacterID}, p.PosX, p.PosY, p.PosZ)
		}
	}
	if target.id != 0 || locked.id != 0 && !locked.mob {
		return target, point, nil
	}
	for i := range s.entities.Mobs {
		m := &s.entities.Mobs[i]
		if m.CurrentHP != 0 && (country == 2 || m.def == nil || m.def.Country != country) &&
			consider(guardTarget{id: m.ID, mob: true}, m.PosX, m.PosY, m.PosZ) {
			mob = m
		}
	}
	return target, point, mob
}

func guardInterval(def *npcDef) time.Duration {
	if def.MoveSpeed <= 0 {
		return 2 * time.Second
	}
	return max(time.Duration(def.MoveSpeed)*time.Millisecond/4, mobNavigationTick)
}

func (n *MapNpc) updateGuard(now time.Time, mapID uint16, s *mapShard, players []PlayerCombatSnapshot, out *npcTick) bool {
	def := findNpcDef(n.Type, n.TypeID)
	if def == nil {
		return false
	}
	if now.Before(n.guardReady) {
		return true
	}
	home := n.patrol[n.patrolIndex]
	point := mapnav.Point{home.X, home.Y, home.Z}
	if !n.guardReturning {
		target, goal, mob := n.findGuardTarget(s, players, def.Country, home, n.guardTarget)
		if target.id != 0 {
			if n.guardTarget.id == 0 {
				if s.guardClaims == nil {
					s.guardClaims = make(map[guardTarget]time.Time)
				}
				s.guardClaims[target] = now.Add(5 * time.Second)
			}
			n.guardTarget = target
			if n.guardChase(now, mapID, s, def, home, target, goal, mob, out) {
				return true
			}
		}
		if n.guardTarget.id == 0 {
			return false
		}
		n.guardTarget, n.guardReturning = guardTarget{}, true
	}
	if DistanceXZ(n.PosX, n.PosZ, home.X, home.Z) <= 1.5 {
		n.guardReturning = false
		return false
	}
	nav, err := loadMobNavigation(mapID)
	if err != nil || !n.guardMove(now, mapID, nav, def, point, 0, out) {
		n.guardReturning = false
		n.guardReady = now.Add(3 * time.Second)
	}
	return true
}

func (n *MapNpc) guardChase(now time.Time, mapID uint16, s *mapShard, def *npcDef, home NpcPatrolPoint, target guardTarget, goal mapnav.Point, mob *MapMob, out *npcTick) bool {
	if DistanceXZ(n.PosX, n.PosZ, home.X, home.Z) > guardLeash {
		return false
	}
	nav, err := loadMobNavigation(mapID)
	if err != nil {
		return false
	}
	reach := float32(guardContact)
	if mapnav.Distance(n.point(), goal) <= reach && (nav == nil || nav.Visible(n.point(), goal)) {
		hit := npcGuardStrike{mapID: mapID, guardID: n.ID, target: target, x: n.PosX, y: n.PosY, z: n.PosZ}
		n.guardTarget, n.guardReturning = guardTarget{}, true
		n.guardReady = now.Add(time.Second)
		if mob != nil {
			hit.hp = mob.CurrentHP
			mob.CurrentHP = 0
			mob.died(now)
			hit.mob = cloneMob(mob)
			n.guardReady = now.Add(500 * time.Millisecond)
		}
		out.strikes = append(out.strikes, hit)
		return true
	}
	return n.guardMove(now, mapID, nav, def, goal, reach, out)
}

func (n *MapNpc) guardMove(now time.Time, mapID uint16, nav *mapnav.Map, def *npcDef, goal mapnav.Point, reach float32, out *npcTick) bool {
	pos := n.point()
	dist := mapnav.Distance(pos, goal)
	step := float32(def.MoveDistance)
	if step <= 0 {
		step = 16 // Stationary guards retain a pursuit gait.
	}
	length := min(step, max(dist-reach*.75, .05))
	var travelled float32
	var next mapnav.Point
	if nav != nil {
		path, ok := nav.Path(pos, goal)
		if !ok {
			return false
		}
		next, travelled, _, ok = nav.Step(pos, path, length, func(x, z float32) bool { return IsMapWalkable(mapID, x, z) })
		if !ok {
			return false
		}
	} else {
		if dist <= .01 {
			return false
		}
		travelled = min(length, dist)
		for i := range next {
			next[i] = pos[i] + (goal[i]-pos[i])*travelled/dist
		}
		if !IsMapWalkable(mapID, next[0], next[2]) {
			return false
		}
	}
	if travelled <= .01 {
		return false
	}
	n.PosX, n.PosY, n.PosZ = next[0], next[1], next[2]
	n.Angle = npcDirection(next[0]-pos[0], next[2]-pos[2])
	n.guardReady = now.Add(max(mobNavigationTick, time.Duration(float64(guardInterval(def))*float64(travelled/step))))
	out.moves = append(out.moves, n.relocate(mapID, mobMotionRun))
	return true
}

// npcDirection is the 0x0E01 angle in whole degrees: the client turns it into
// the heading (-sin, cos) of angle*pi/180 (Game.exe 0x5659a0).
func npcDirection(dx, dz float32) uint16 {
	degrees := math.Round(math.Atan2(-float64(dx), float64(dz)) * 180 / math.Pi)
	if degrees < 0 {
		degrees += 360
	}
	return uint16(int(degrees) % 360)
}

func buildNpcAttack(npcID, target uint32, mob bool, hp uint32) []byte {
	op := packet.MapNpcAttackPlayer
	if mob {
		op = packet.MapNpcAttackMob
	}
	w := packet.New(op)
	w.U8(1)
	w.U32(npcID)
	w.U32(target)
	w.U16(uint16(min(hp, 65535)))
	return w.MustFinish()
}

func dispatchNpcStrike(hit npcGuardStrike, log logPrinter) {
	if hit.target.mob {
		sendView(hit.mapID, hit.x, hit.z, buildNpcAttack(hit.guardID, hit.target.id, true, hit.hp), "0x0E06 guard attacks mob")
		sendView(hit.mapID, hit.mob.PosX, hit.mob.PosZ, buildMobDeathBy(hit.mob.ID, hit.guardID, deathTypeGuard), "guard mob death")
		eventMobDestroyed(nil, hit.mapID, &hit.mob, log)
	} else if player := FindPlayerByID(hit.target.id); player != nil && player.MapID() == hit.mapID {
		player.QueueDamage(pendingDamage{Guard: &hit})
	}
}

// The session owns player death. Revalidate map and contact after crossing its
// mailbox so a teleport cannot apply a stale strike to a new position.
func (s *Session) applyGuardStrike(hit npcGuardStrike) bool {
	ch := s.char
	if ch == nil || s.dead || ch.MapID != hit.mapID ||
		mapnav.Distance(mapnav.Point{ch.PosX, ch.PosY, ch.PosZ}, mapnav.Point{hit.x, hit.y, hit.z}) > guardContact {
		return false
	}
	damage := s.stats().CurrentHP
	if damage == 0 {
		return false
	}
	s.lastCombatAt = time.Now()
	s.cancelMove()
	s.breakStealth()
	s.cancelTravelCast()
	s.cancelVehicleCast()
	ch.CurrentHP = 0
	s.damageLedger = nil
	s.persist()
	stats := s.stats()
	s.send(buildCurrentHitpoints(&stats), "guard death pools")
	sendView(hit.mapID, hit.x, hit.z, buildNpcAttack(hit.guardID, ch.ID, false, damage), "0x0E05 guard attacks character")
	state, death := buildCharacterDeathState(ch.ID), buildCharacterDeath(ch.ID, deathTypeGuard, hit.guardID)
	s.send(state, "guard death state")
	s.send(death, "guard death")
	s.sendToViewers(state, "guard death state (view)")
	s.sendToViewers(death, "guard death (view)")
	if s.player != nil {
		s.player.SetAlive(false)
		duelLeave(s.player, 3, true)
		guildDuelLeave(s.player, nil, 3)
	}
	return true
}
