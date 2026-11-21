package world

import (
	"math/rand/v2"
	"time"

	"github.com/Spelunkern/Shaiya-GX/server/internal/db"
	"github.com/Spelunkern/Shaiya-GX/server/internal/packet"
)

// TypeEffect 6 skills are CUser::SkillOnce 0x00453610 (reached through
// UseSkillUser / AttackUser), switched on the skill's Type (TypeDetail).
const (
	typeEffectOnce = 6

	detailLifeSkill = 121 // resurrection: revive a dead user (LifeSkill)
	detailKillSoul  = 122 // send an enemy corpse to its bind point
	detailDetect    = 123 // reveal an enemy's defence element, level, mode
	detailCheck     = 124 // success roll only
	detailReturn    = 125 // move the caster to its bind point

	skillResultFailed    = 3
	skillResultNotTarget = 11
)

// pendingLifeSkill asks a dead user's session to revive at the caster.
type pendingLifeSkill struct {
	CasterID uint32
	X, Y, Z  float32
}

func (p *ConnectedPlayer) QueueLifeSkill(l pendingLifeSkill) {
	p.mailMu.Lock()
	p.lifeSkill = &l
	p.mailMu.Unlock()
}

func (p *ConnectedPlayer) TakeLifeSkill() *pendingLifeSkill {
	p.mailMu.Lock()
	defer p.mailMu.Unlock()
	l := p.lifeSkill
	p.lifeSkill = nil
	return l
}

type corpseTarget struct {
	player *ConnectedPlayer
	state  uint64
}

type pendingSoulKill struct {
	source *ConnectedPlayer
	state  uint64
	mapID  uint16
	skill  db.CharacterSkill
}

func (p *ConnectedPlayer) queueSoulKill(k pendingSoulKill) bool {
	p.mailMu.Lock()
	defer p.mailMu.Unlock()
	if p.lifeState.Load() != k.state || p.soulKill != nil && p.soulKill.state == k.state {
		return false
	}
	p.soulKill = &k
	return true
}

func (p *ConnectedPlayer) takeSoulKill() *pendingSoulKill {
	p.mailMu.Lock()
	defer p.mailMu.Unlock()
	k := p.soulKill
	p.soulKill = nil
	return k
}

// Soul Burn follows the normal PvP team and visibility rules with a dead target.
func (s *Session) soulBurnCorpse(p *ConnectedPlayer) corpseTarget {
	if p == nil || s.player == nil || s.dead || p == s.player {
		return corpseTarget{}
	}
	state := p.lifeState.Load()
	if state == 0 || state&1 != 0 || s.isTeam(p) {
		return corpseTarget{}
	}
	return corpseTarget{player: p, state: state}
}

func soulBurnRejection(source uint32, k db.CharacterSkill) []byte {
	return buildSkillUseResult(packet.SkillUseNoTarget, skillResultNotTarget, source, 0, k.SkillID, k.Level, 0, 0, 0)
}

func (s *Session) rejectSoulBurn(k db.CharacterSkill) {
	s.send(soulBurnRejection(s.char.ID, k), "0x0511 Soul Burn target refused")
}

// equipmentDefenseAttr reads CUser +0x1284: live authoritative defense
// element, including application/removal events; equipment-only before entry.
func (p *ConnectedPlayer) equipmentDefenseAttr() uint8 {
	p.mu.Lock()
	items := append([]db.Item(nil), p.items...)
	known, attr := p.elementKnown, p.defenseElement
	p.mu.Unlock()
	if known {
		return attr
	}
	return GetEquipmentStats(items).DefenseAttr
}

// buildDetectInfo (0x0558): target defence element u32, level u16, mode u8.
func buildDetectInfo(attr uint32, level uint16) []byte {
	w := packet.New(packet.SkillDetectInfo)
	w.U32(attr)
	w.U16(level)
	w.U8(0)
	return w.MustFinish()
}

// userSkillTarget resolves a TypeEffect 6 target: the caster itself for
// self skills (TargetType 2), else a user on the caster's map.
func (s *Session) userSkillTarget(def *db.SkillDef, target uint32) (*ConnectedPlayer, bool) {
	if def.TargetType == targetTypeSelf || target == s.char.ID {
		return s.player, s.player != nil
	}
	p := FindPlayerByID(target)
	if p == nil || p.MapID() != s.char.MapID || !s.player.canSee(p) || p.immune.Load() {
		return nil, false
	}
	return p, true
}

// inUserSkillRange checks launch reach and visibility without release grace.
func (s *Session) inUserSkillRange(p *ConnectedPlayer, def *db.SkillDef) bool {
	return s.userSkillReach(p, def, false)
}

// skillOnceSucceeds is CUser::AttackSuccess 0x00456b10 for these skills:
// a block (ChkBlock; checked when the caster is the target) fails it, then
// SuccessType 1 rolls SuccessValue%. Types 2 and 3 add 100 x
// caster/(caster+target) LUC or WIS in integer division, which is 0 unless
// the target's stat is 0, so they roll SuccessValue% too. The level-based
// default and the state resistances are not modelled: every EP4.5
// TypeEffect 6 skill is SuccessType 1.
func (s *Session) skillOnceSucceeds(def *db.SkillDef, self bool) bool {
	if self && s.blocksAttack(int(def.Kind)) {
		return false
	}
	switch def.SuccessType {
	case 1, 2, 3:
		return rand.IntN(100) < int(def.SuccessValue)
	}
	return true
}

// skillOnce runs a TypeEffect 6 skill after its cast: SP/MP and the reset
// time are spent whatever the outcome (AttackUser 0x00451d1d).
func (s *Session) skillOnce(k db.CharacterSkill, def *db.SkillDef, target uint32, stats CharacterStats, cast *pendingCast) {
	ch := s.char
	p, ok := s.userSkillTarget(def, target)
	var corpse corpseTarget
	if def.TypeDetail == detailKillSoul {
		corpse = s.soulBurnCorpse(p)
		if corpse.player == nil || cast != nil && cast.corpse != corpse {
			s.rejectSoulBurn(k)
			return
		}
	}
	if !ok || !s.userSkillReach(p, def, cast != nil) {
		s.sendUserSkillRangeFailure(k, p)
		return
	}
	s.startCooldown(k, def, cast != nil)
	sp, mp := s.skillCosts(def, &stats)
	s.spendDelta(mp, sp)
	s.persist()
	s.send(buildSkillUsedSpMp(sp, mp), "0x050C skill used sp/mp")
	result := func(code uint8, targetID uint32) {
		s.broadcastNear(buildSkillUseResult(packet.SkillUseNoTarget, code, ch.ID, targetID, k.SkillID, k.Level, 0, 0, 0),
			"0x0511 skill once", true)
	}
	self := p == s.player
	switch def.TypeDetail {
	case detailLifeSkill:
		if p.IsAlive() {
			s.send(buildSkillUseResult(packet.SkillUseNoTarget, skillResultNotTarget, ch.ID, 0, k.SkillID, k.Level, 0, 0, 0),
				"0x0511 resurrection target alive")
			return
		}
		if !s.skillOnceSucceeds(def, self) {
			result(skillResultFailed, p.CharacterID)
			return
		}
		result(skillResultNormal, p.CharacterID)
		p.QueueLifeSkill(pendingLifeSkill{ch.ID, ch.PosX, ch.PosY, ch.PosZ})
	case detailKillSoul:
		if !s.skillOnceSucceeds(def, false) {
			result(skillResultFailed, p.CharacterID)
		} else if !p.queueSoulKill(pendingSoulKill{source: s.player, state: corpse.state, mapID: ch.MapID, skill: k}) {
			s.rejectSoulBurn(k)
		}
	case detailDetect:
		if !s.skillOnceSucceeds(def, self) {
			result(skillResultFailed, p.CharacterID)
			return
		}
		if p.Faction() == s.faction {
			result(skillResultNotTarget, p.CharacterID)
			return
		}
		result(skillResultNormal, p.CharacterID)
		t := p.Snapshot()
		s.send(buildDetectInfo(uint32(p.equipmentDefenseAttr()), t.Level), "0x0558 detect info")
	case detailCheck:
		if !s.skillOnceSucceeds(def, self) {
			result(skillResultFailed, p.CharacterID)
			return
		}
		result(skillResultNormal, p.CharacterID)
	case detailReturn:
		if s.dead {
			return
		}
		result(skillResultNormal, 0)
		m, x, y, z := s.respawnDestination(ch)
		// UpdateResetPosition moves the user on the next tick.
		s.player.QueueTeleport(m, x, y, z)
	default:
		s.log.Printf("GX Server World skill once with unknown type %d skill_id=%d", def.TypeDetail, k.SkillID)
	}
	s.log.Printf("GX Server World skill once skill_id=%d type=%d target=%d", k.SkillID, def.TypeDetail, target)
}

// tickSkillOnce applies queued skill effects on this (dead) user.
func (s *Session) tickSkillOnce() {
	p := s.player
	if p == nil {
		return
	}
	if l := p.TakeLifeSkill(); l != nil && s.dead && !s.invulnerable() {
		s.lifeSkill(l)
	}
	if k := p.takeSoulKill(); k != nil {
		ch := s.char
		if !s.dead || s.invulnerable() || ch.MapID != k.mapID || p.lifeState.Load() != k.state {
			k.source.Send(soulBurnRejection(k.source.CharacterID, k.skill), "0x0511 Soul Burn corpse no longer available")
			return
		}
		// The corpse is still present when the result arrives. Its view sees
		// the impact before corpse leave and rebirth packets.
		frame := buildSkillUseResult(packet.SkillUseNoTarget, skillResultNormal, k.source.CharacterID, ch.ID, k.skill.SkillID, k.skill.Level, 0, 0, 0)
		sendNear(ch.MapID, ch.PosX, ch.PosZ, CombatRadius, frame, "0x0511 Soul Burn")
		s.rebirthAtBind()
	}
}

// lifeSkill is CUser::LifeSkill 0x00459d20: the corpse leaves the view,
// 0x0551 kind 4 with the user's exp and the caster's position goes to the
// view, the pools are full, no exp is lost.
func (s *Session) lifeSkill(l *pendingLifeSkill) {
	ch := s.char
	p, ok := s.groundTrustedDestination(l.X, l.Y, l.Z)
	if !ok {
		return
	}
	s.broadcastNear(buildCharacterReviveInit(ch.ID), "0x0406 corpse leave", true)
	s.refillPools()
	s.dead, s.rebirthAt, s.rebirthParty = false, time.Time{}, false
	ch.PosX, ch.PosY, ch.PosZ = p[0], p[1], p[2]
	if s.movement.geometry != nil {
		s.movement.at = time.Time{}
		_ = s.initPlayerMovement(time.Now())
	}
	if s.player != nil {
		s.player.SetAlive(true)
		s.player.UpdateCharacter(*ch)
	}
	s.partyMove()
	s.persist()
	s.placeView()
	s.broadcastNear(buildCharacterResurrect(ch.ID, rebirthKindBind, ch.Exp, ch.PosX, ch.PosY, ch.PosZ), "0x0551 life skill", true)
	st := s.stats()
	s.send(buildCharacterReviveHitpoints(ch.ID, st.CurrentHP, st.CurrentMP, st.CurrentSP), "0x0505 life skill pools")
	s.log.Printf("GX Server World revived by skill character_id=%d caster=%d", ch.ID, l.CasterID)
}
