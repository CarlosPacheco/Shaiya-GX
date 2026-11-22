package world

import (
	"time"

	"github.com/Spelunkern/Shaiya-GX/internal/proto"
)

// Resurrection follows ps_game.exe: the 0x0553 request (CUser::PacketPC),
// CUser::KillSoul 0x004599e0, KillSoulByItem 0x00459740,
// KillSoulByItemNoMove 0x004598a0, KillSoulParty 0x004596a0,
// UpdateRebirth 0x0048d240, FindItemRebirth 0x0045f740 and Rebirth
// 0x00459ec0.

// 0x0553 request byte.
const (
	reviveAtBind     = 2
	reviveWithParty  = 3
	reviveRuneAtBind = 4
	reviveRuneHere   = 5
)

// CUser::Rebirth kind byte of 0x0551.
const (
	rebirthKindBind  = 4
	rebirthKindParty = 5
)

const (
	// rebirthDeadline is EnterDead's tick + 200000: the dead user is
	// revived at the bind point when it has not chosen.
	rebirthDeadline  = 200 * time.Second
	partyRebirthWait = proto.PartyReviveSeconds * time.Second
)

// startRebirthTimer is EnterDead's deadline (code 0).
func (s *Session) startRebirthTimer() {
	s.rebirthAt, s.rebirthParty = time.Now().Add(rebirthDeadline), false
}

// handleResurrect (0x0553): mode byte.
func (s *Session) handleResurrect(p []byte) {
	if len(p) < 3 || s.char == nil || !s.dead {
		return
	}
	switch p[2] {
	case reviveAtBind, reviveRuneAtBind, reviveRuneHere:
		s.rebirthAtBind()
	case reviveWithParty:
		if !s.inParty() {
			s.rebirthAtBind()
			return
		}
		s.rebirthAt, s.rebirthParty = time.Now().Add(partyRebirthWait), true
	}
}

// tickRebirth is UpdateRebirth's death timer.
func (s *Session) tickRebirth(now time.Time) {
	if !s.dead || s.rebirthAt.IsZero() || now.Before(s.rebirthAt) {
		return
	}
	if s.rebirthParty && s.inParty() {
		s.killSoulParty()
		return
	}
	s.rebirthAtBind()
}

// rebirthAtBind applies the normal death penalty and returns the user with full pools.
func (s *Session) rebirthAtBind() {
	ch := s.char
	lost := expDrop(ch, s.deathLoss, s.faction)
	m, x, y, z := s.respawnDestination(ch)
	s.rebirth(m, x, y, z, rebirthKindBind, true, lost)
}

// killSoulParty is KillSoulParty: at the party leader's position with
// full pools (after the exp loss) when the leader is alive on the same
// map; otherwise return to the bind point.
func (s *Session) killSoulParty() {
	ch := s.char
	l := s.partyLeader()
	if l == nil || l == s.player || !l.IsAlive() || l.MapID() != ch.MapID {
		s.rebirthAtBind()
		return
	}
	lost := expDrop(ch, s.deathLoss, s.faction)
	pos := l.Snapshot()
	s.rebirth(ch.MapID, pos.PosX, pos.PosY, pos.PosZ, rebirthKindParty, true, lost)
}

func (s *Session) inParty() bool {
	if s.player == nil {
		return false
	}
	partyMu.Lock()
	defer partyMu.Unlock()
	return partyOf(s.player) != nil
}

func (s *Session) partyLeader() *ConnectedPlayer {
	if s.player == nil {
		return nil
	}
	partyMu.Lock()
	defer partyMu.Unlock()
	if pt := partyOf(s.player); pt != nil {
		return pt.leaderPlayer()
	}
	return nil
}

// refillPools sets current HP, MP and SP to their effective maxima.
func (s *Session) refillPools() {
	ch := s.char
	full := StatsFor(ch, s.equipment())
	ch.CurrentHP, ch.CurrentMP, ch.CurrentSP = full.MaxHP, full.MaxMP, full.MaxSP
}

// rebirth restores the character and publishes the result to the world.
func (s *Session) rebirth(mapID uint16, x, y, z float32, kind uint8, restore bool, lost uint32) {
	ch := s.char
	if !s.canEnterMap(mapID, ch.Level) {
		birth := birthPosition(ch)
		mapID, x, y, z = birth.MapID, birth.X, birth.Y, birth.Z
	}
	if mapID == ch.MapID {
		p, ok := s.groundTrustedDestination(x, y, z)
		if !ok {
			return
		}
		x, y, z = p[0], p[1], p[2]
	}
	if restore {
		s.refillPools()
	}
	s.broadcastNear(buildCharacterReviveInit(ch.ID), "0x0406 corpse leave", true)
	s.dead, s.rebirthAt, s.rebirthParty = false, time.Time{}, false
	if s.player != nil {
		s.player.SetAlive(true)
	}
	s.persist()
	if mapID == ch.MapID {
		if x != ch.PosX || y != ch.PosY || z != ch.PosZ {
			s.clearView(true)
			ch.PosX, ch.PosY, ch.PosZ = x, y, z
			if s.movement.geometry != nil {
				s.movement.at = time.Time{}
				_ = s.initPlayerMovement(time.Now())
			}
			if s.player != nil {
				s.player.UpdateCharacter(*ch)
				s.player.UpdateMotion(0, 0)
			}
			s.savePosition()
			s.placeView()
		} else if s.player != nil {
			s.player.UpdateCharacter(*ch)
		}
		s.broadcastNear(buildCharacterResurrect(ch.ID, kind, ch.Exp, x, y, z), "0x0551 rebirth", true)
	} else {
		// 0x0551 has coordinates but no map ID. Publish the map change first
		// so a client never applies the destination to the old scene.
		s.applyTeleport(mapID, x, y, z, "rebirth")
		s.send(buildCharacterResurrect(ch.ID, kind, ch.Exp, ch.PosX, ch.PosY, ch.PosZ), "0x0551 rebirth")
	}
	stats := s.stats()
	s.send(buildCharacterReviveHitpoints(ch.ID, stats.CurrentHP, stats.CurrentMP, stats.CurrentSP), "0x0505 rebirth pools")
	s.syncPlayerSpeed(true)
	s.log.Printf("GX Server World rebirth character_id=%d kind=%d map=%d pos=(%.2f, %.2f, %.2f) exp_lost=%d",
		ch.ID, kind, mapID, x, y, z, lost)
}

// reviveOnExit settles a dead character that leaves the game, or that an older
// save kept at 0 HP: it is revived at its bind point with full pools and the
// usual experience loss, so it re-enters alive and the death is not kept.
func (s *Session) reviveOnExit(loss int) {
	ch := s.char
	if ch == nil || !s.dead && ch.CurrentHP != 0 {
		return
	}
	lost := expDrop(ch, loss, s.faction)
	m, x, y, z := s.respawnDestination(ch)
	if !s.canEnterMap(m, ch.Level) {
		birth := birthPosition(ch)
		m, x, y, z = birth.MapID, birth.X, birth.Y, birth.Z
	}
	ch.MapID, ch.PosX, ch.PosY, ch.PosZ = m, x, y, z
	s.refillPools()
	s.dead, s.rebirthAt, s.rebirthParty = false, time.Time{}, false
	if s.player != nil {
		s.player.SetAlive(true)
	}
	s.log.Printf("GX Server World revived at bind point on exit character_id=%d map=%d pos=(%.2f, %.2f, %.2f) exp_lost=%d", ch.ID, m, x, y, z, lost)
}

// exitDeathLoss is the experience percent a dead character pays when it leaves.
func (s *Session) exitDeathLoss() int {
	if s.dead {
		return s.deathLoss
	}
	return s.deathExpLoss()
}
