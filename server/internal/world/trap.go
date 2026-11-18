package world

import (
	"math"
	"time"

	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/server/internal/db"
	"github.com/Spelunkern/Shaiya-GX/server/internal/packet"
)

const (
	typeEffectField = 5   // CUser::SkillField: a trap or field on the ground
	detailTrap      = 101 // springs once on the first enemy, then it is gone
	detailFireBush  = 102 // a curtain of fire: fixed damage to every enemy in it, every half second
	fireBushTick    = 500 * time.Millisecond
	trapSweep       = 250 * time.Millisecond
	trapDamageMax   = 30000
)

// placedTrap is a CTrab: owner, position, skill and the time it ends. It is
// invisible to everybody; only its cast and its hit are announced.
type placedTrap struct {
	mapID   uint16
	x, y, z float32
	def     *db.SkillDef
	endsAt  time.Time
	nextHit time.Time // Fire Bush: the next pulse; zero for a trap
}

// skillField is CUser::SkillField (AttackUser, TypeEffect 5): the trap is left
// on the ground where the caster stands, lasts the skill's KeepTime and is
// announced like any cast (0x0511, result 0, no target). There is no limit of
// traps: the skill's own reuse time is the only one.
func (s *Session) skillField(k db.CharacterSkill, def *db.SkillDef, stats CharacterStats, completed ...bool) {
	if def.TypeDetail != detailTrap && def.TypeDetail != detailFireBush {
		s.log.Printf("GX Server World rejected skill use: field detail %d not modeled skill_id=%d", def.TypeDetail, k.SkillID)
		return
	}
	ch := s.char
	s.startCooldown(k, def, len(completed) > 0 && completed[0])
	sp, mp := s.skillCosts(def, &stats)
	s.spendDelta(mp, sp)
	s.persist()
	s.send(buildSkillUsedSpMp(sp, mp), "0x050C skill used sp/mp")
	s.broadcastNear(buildSkillUseResult(packet.SkillUseNoTarget, 0, ch.ID, 0, k.SkillID, k.Level, 0, 0, 0), "0x0511 trap placed", true)
	t := placedTrap{mapID: ch.MapID, x: ch.PosX, y: ch.PosY, z: ch.PosZ, def: def,
		endsAt: time.Now().Add(time.Duration(def.KeepSeconds) * time.Second)}
	if def.TypeDetail == detailFireBush {
		t.nextHit = time.Now()
		// Everybody around sees the curtain of fire; a trap is invisible and is not announced.
		w := packet.New(packet.Opcode(proto.GXField))
		w.U32(ch.ID)
		w.U16(k.SkillID)
		w.U8(k.Level)
		w.F32(ch.PosX)
		w.F32(ch.PosY)
		w.F32(ch.PosZ)
		w.U16(uint16(min(def.KeepSeconds, 65535)))
		sendNear(ch.MapID, ch.PosX, ch.PosZ, CombatRadius, w.MustFinish(), "GX field placed")
	}
	s.traps = append(s.traps, t)
	s.log.Printf("GX Server World trap placed skill_id=%d level=%d radius=%d seconds=%d", k.SkillID, k.Level, def.ApplyRange, def.KeepSeconds)
}

// trapDamage is TrabDamage 0x00422480. DamageType 4 takes the value as a
// percentage of the target's maximum pool and 5 of its current pool; any other
// type is the value itself. Each non-zero part is at least 1 and at most 30000.
func trapDamage(def *db.SkillDef, current, maximum [3]uint32) [3]uint32 {
	values := [3]int32{def.DamageHP, def.DamageSP, def.DamageMP}
	var out [3]uint32
	for i, v := range values {
		if v == 0 {
			continue
		}
		switch def.DamageType {
		case 4:
			out[i] = uint32(max(int64(float64(v)/100*float64(maximum[i])), 1))
		case 5:
			out[i] = uint32(max(int64(float64(v)/100*float64(current[i])), 1))
		default:
			out[i] = uint32(max(v, 0))
			continue
		}
		out[i] = min(out[i], trapDamageMax)
	}
	return out
}

// tickTraps is CZone::UpdateTrab for this user's traps: an expired trap goes
// away, and the first enemy (user or mob) within ApplyRange of a trap springs
// it. A trap left in another map is dropped.
func (s *Session) tickTraps(now time.Time) {
	if len(s.traps) == 0 || s.char == nil || now.Before(s.nextTrapAt) {
		return
	}
	s.nextTrapAt = now.Add(trapSweep) // finer than the half second of Fire Bush
	kept := s.traps[:0]
	for _, t := range s.traps {
		if !now.Before(t.endsAt) || t.mapID != s.char.MapID {
			continue
		}
		if t.def.TypeDetail == detailFireBush {
			// The curtain of fire stays for its whole time and burns every half second.
			if !now.Before(t.nextHit) {
				s.fixedDamagePulse(t.def, t.x, t.y, t.z)
				t.nextHit = t.nextHit.Add(fireBushTick)
				if t.nextHit.Before(now) {
					t.nextHit = now.Add(fireBushTick)
				}
			}
		} else if s.springTrap(t) {
			continue
		}
		kept = append(kept, t)
	}
	s.traps = kept
}

// springTrap hits the nearest enemy in reach and reports whether it did.
func (s *Session) springTrap(t placedTrap) bool {
	ch := s.char
	radius := float32(t.def.ApplyRange)
	var foe *ConnectedPlayer
	var mobID uint32
	best := float32(-1)
	for _, p := range s.hostileUsersWithin(t.x, t.z, radius) {
		if !p.GMVisible() {
			continue
		}
		px, pz := p.PositionXZ()
		if d := DistanceXZ(t.x, t.z, px, pz); best < 0 || d < best {
			best, foe, mobID = d, p, 0
		}
	}
	for _, id := range s.mobsNear(t.x, t.z, radius) {
		mob, ok := MobByID(ch.MapID, id)
		md := db.MustDefs().Mobs[mob.MobID]
		if !ok || md == nil || mobFriendly(md, s.faction) {
			continue
		}
		if t.def.StateType != 0 && md.ResistState[t.def.StateType-1] != 0 {
			continue // "some monsters have immunity"
		}
		if d := DistanceXZ(t.x, t.z, mob.PosX, mob.PosZ); best < 0 || d < best {
			best, foe, mobID = d, nil, id
		}
	}
	switch {
	case foe != nil:
		foe.QueueUserHit(pendingUserHit{AttackerID: ch.ID, Trap: true, Skill: t.def})
	case mobID != 0:
		s.trapHitsMob(mobID, t.def)
	default:
		return false
	}
	s.lastCombatAt = time.Now()
	s.log.Printf("GX Server World trap sprang skill_id=%d level=%d", t.def.SkillID, t.def.SkillLevel)
	return true
}

// trapHitsMob is the mob branch of UpdateTrab: TrabDamage on the mob's pools,
// CMob::Damage with the skill's Arrow as aggro, 0x0519 to the view, and the
// mob turns to the owner when it survives.
func (s *Session) trapHitsMob(id uint32, def *db.SkillDef) {
	ch := s.char
	mob, ok := MobByID(ch.MapID, id)
	md := db.MustDefs().Mobs[mob.MobID]
	if !ok || md == nil {
		return
	}
	d := trapDamage(def, [3]uint32{mob.CurrentHP, md.MaxSP, md.MaxMP}, [3]uint32{mob.MaxHP, md.MaxSP, md.MaxMP})
	r, ok := DamageMobFrom(ch.MapID, id, d[0], ch.ID)
	if !ok {
		return
	}
	AddMobAggro(ch.MapID, id, ch.ID, int64(def.Aggro))
	words := [3]uint16{uint16(min(r.DamageHP, 65535)), uint16(min(d[1], 65535)), uint16(min(d[2], 65535))}
	sendNear(ch.MapID, r.Mob.PosX, r.Mob.PosZ, CombatRadius,
		buildSkillUseResult(opSkillOther, 0, ch.ID, r.Mob.ID, def.SkillID, def.SkillLevel, words[0], words[1], words[2]), "0x0519 trap damage")
	if s.resolveMobDamage(&r) {
		ForceMobTarget(ch.MapID, id, ch.ID)
	}
}

// applyTrapHit is the victim side of a sprung trap: its stealth ends
// (RemApplySkillHide), TrabDamage on its pools, CUser::Damage (HP, SP and MP),
// 0x0513 to the view; true when this character died.
func (s *Session) applyTrapHit(h pendingUserHit) bool {
	ch := s.char
	s.breakStealth()
	st := s.stats()
	d := trapDamage(h.Skill, [3]uint32{st.CurrentHP, st.CurrentSP, st.CurrentMP}, [3]uint32{st.MaxHP, st.MaxSP, st.MaxMP})
	killed := s.takeUserSkillDamage(h.AttackerID, d)
	shown := d
	if d[0] > 0 { // Health Assistant moved part of it to SP / MP
		shown = [3]uint32{s.assisted[0], d[1] + s.assisted[1], d[2] + s.assisted[2]}
	}
	sendNear(ch.MapID, ch.PosX, ch.PosZ, CombatRadius,
		buildSkillUseResult(opUserSkillOther, 0, h.AttackerID, ch.ID, h.Skill.SkillID, h.Skill.SkillLevel,
			uint16(min(shown[0], 65535)), uint16(min(shown[1], 65535)), uint16(min(shown[2], 65535))), "0x0513 trap damage")
	s.afterUserDamage(h.AttackerID, killed)
	return killed
}

// mobsNear lists the living mobs whose centre is within r of a point, measured
// on the ground plane. The original area loops (UpdateTrab, TauntMob) use the
// distance in X and Z only, so a slope, a mob's height or a wall between them
// never narrows a fire's or a trap's reach.
//
// An optional height (y, reach) also keeps only the mobs within reach of it
// vertically.
func (s *Session) mobsNear(x, z, r float32, height ...float32) []uint32 {
	var out []uint32
	for _, m := range EntitiesOnMap(s.char.MapID).Mobs {
		if len(height) == 2 && float32(math.Abs(float64(m.PosY-height[0]))) > height[1] {
			continue
		}
		if m.CurrentHP > 0 && DistanceXZ(x, z, m.PosX, m.PosZ) <= r {
			out = append(out, m.ID)
		}
	}
	return out
}
