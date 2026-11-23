package world

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// gmMaxEraseRange bounds /mera <range>, in world units.
const gmMaxEraseRange = 1000

// gmTargetID reads the "#<id>" the client writes in place of "t".
func gmTargetID(args []string) (uint32, bool, error) {
	if len(args) != 1 || !strings.HasPrefix(args[0], "#") {
		return 0, false, nil
	}
	id, err := strconv.ParseUint(args[0][1:], 10, 32)
	if err != nil || id == 0 {
		return 0, true, fmt.Errorf("target")
	}
	return uint32(id), true, nil
}

// MobIDsWithin lists the living mobs of a map within radius units of (x, z).
func MobIDsWithin(mapID uint16, x, z, radius float32) []uint32 {
	sh := shard(mapID)
	sh.mu.Lock()
	defer sh.mu.Unlock()
	var out []uint32
	for i := range sh.entities.Mobs {
		m := &sh.entities.Mobs[i]
		if m.CurrentHP > 0 && DistanceXZ(x, z, m.PosX, m.PosZ) <= radius {
			out = append(out, m.ID)
		}
	}
	return out
}

// eraseMob kills a mob with no credit, drop or experience; its common respawn
// still follows. False when the mob cannot be killed (already dead, resetting
// home or scripted invulnerable).
func (s *Session) eraseMob(id uint32) bool {
	mapID := s.char.MapID
	m, ok := MobByID(mapID, id)
	if !ok {
		return false
	}
	result, ok := DamageMobFrom(mapID, id, m.CurrentHP, 0)
	if !ok || !result.Killed {
		return false
	}
	sendNear(mapID, result.Mob.PosX, result.Mob.PosZ, CombatRadius, buildMobDeath(result.Mob.ID, 0), "0x0606 mob death (GM erase)")
	return true
}

// gmEraseMobs is /mera on the selected target (the client sends #<id>) or /mera <range>.
func (s *Session) gmEraseMobs(args []string) error {
	if id, target, err := gmTargetID(args); target {
		if err != nil {
			return err
		}
		if !s.eraseMob(id) {
			s.gmReply("GM: that target is not a living monster.")
			return nil
		}
		s.gmReply("GM: monster erased.")
		return nil
	}
	n, err := gmNumbers(args, gmMaxEraseRange)
	if err != nil {
		s.gmReply("Usage: /mera (your target) | /mera <range 1-1000>")
		return nil
	}
	ch := s.char
	count := 0
	for _, id := range MobIDsWithin(ch.MapID, ch.PosX, ch.PosZ, float32(n[0])) {
		if s.eraseMob(id) {
			count++
		}
	}
	s.gmReply(fmt.Sprintf("GM: %d monsters erased.", count))
	return nil
}

// gmEraseNPC is /nera t: the NPC leaves its map until the server restarts.
func (s *Session) gmEraseNPC(args []string) error {
	id, target, err := gmTargetID(args)
	if !target || err != nil {
		s.gmReply("Usage: /nera (your target)")
		return nil
	}
	sh := shard(s.char.MapID)
	sh.mu.Lock()
	var gone *MapNpc
	for i := range sh.entities.NPCs {
		if sh.entities.NPCs[i].ID == id {
			n := sh.entities.NPCs[i]
			gone = &n
			sh.entities.NPCs = append(sh.entities.NPCs[:i], sh.entities.NPCs[i+1:]...)
			sh.npcRevision.Add(1)
			break
		}
	}
	sh.mu.Unlock()
	if gone == nil {
		s.gmReply("GM: that target is not an NPC.")
		return nil
	}
	for _, p := range PlayersViewing(s.char.MapID, gone.cell) {
		viewLeave(p, viewNpc, gone.ID, buildNpcLeave(gone.ID), "GM NPC erase")
	}
	s.gmReply("GM: NPC removed until the next server restart.")
	return nil
}

// gmKill is /kill t: a legitimate, immediate kill of a monster or a player of
// any faction. A monster drops and rewards its killer as usual.
func (s *Session) gmKill(args []string) error {
	id, target, err := gmTargetID(args)
	if !target || err != nil {
		s.gmReply("Usage: /kill (your target)")
		return nil
	}
	ch := s.char
	if id == ch.ID {
		s.gmReply("GM: choose another target.")
		return nil
	}
	if m, ok := MobByID(ch.MapID, id); ok {
		result, ok := DamageMobFrom(ch.MapID, id, m.CurrentHP, ch.ID)
		if !ok || !result.Killed {
			s.gmReply("GM: that monster cannot be killed.")
			return nil
		}
		AddMobAggro(ch.MapID, id, ch.ID, 0)
		s.resolveMobDamage(&result)
		return nil
	}
	if p := FindPlayerByID(id); p != nil && p.MapID() == ch.MapID && p.IsAlive() {
		p.QueueUserHit(pendingUserHit{AttackerID: ch.ID, Execute: true})
		s.gmReply("GM: target executed.")
		return nil
	}
	s.gmReply("GM: that target cannot be killed.")
	return nil
}

// applyExecute is the target side of /kill on a player: all its health is
// lost whatever its immunity, and the death is credited to the GM like any
// player kill.
func (s *Session) applyExecute(h pendingUserHit) bool {
	ch := s.char
	if ch == nil {
		return false
	}
	cur := s.stats()
	if cur.CurrentHP == 0 {
		return false
	}
	s.lastCombatAt = time.Now()
	s.assisted = [3]uint32{cur.CurrentHP}
	s.cancelMove()
	s.breakStealth()
	s.cancelTravelCast()
	s.addDamage(damageByUser, h.AttackerID, cur.CurrentHP)
	ch.CurrentHP = 0
	s.persist()
	s.cancelVehicleCast()
	sendNear(ch.MapID, ch.PosX, ch.PosZ, CombatRadius,
		buildCharacterUserAttack(h.AttackerID, ch.ID, attackSuccessNormal, uint16(min(cur.CurrentHP, 65535))), "0x0502 user attack (GM kill)")
	s.afterUserDamage(h.AttackerID, true)
	return true
}
