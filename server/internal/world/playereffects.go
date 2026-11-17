package world

import (
	"math/rand/v2"
	"time"

	"github.com/Spelunkern/Shaiya-GX/internal/attackspeed"
	"github.com/Spelunkern/Shaiya-GX/internal/movespeed"

	"github.com/Spelunkern/Shaiya-GX/server/internal/db"
	"github.com/Spelunkern/Shaiya-GX/server/internal/packet"
)

// Skills applied to the player: buffs (TypeEffect 4) and mob debuffs (3),
// as CUser::AddApplySkill (0x00484850), AddApplySkillBuff / Debuff,
// ChkKeepSkill (0x00487190, every 3 s) and ChkEndTimeSkill run them.
// Debuff details: 61 damage over time (0x050F), 62/67 ability loss,
// 63 whole-HP hit on expiry, 64/66 no actions, 65 no movement (0x051C
// speeds, 255 when blocked; 66 also ends on damage), 68 no armour element.
// Buff details 85/86 block a category's attacks by count or chance and 92
// blocks everything (VerticalMobsActorAttackBlock).

// addEffect starts (or replaces) an applied skill and sends 0x050D. The
// same skill at a lower level does not replace a higher one.
func (s *Session) addEffect(def *db.SkillDef, casterMob uint32, sourceUser ...uint32) (ActiveBuff, bool) {
	if !s.canAddEffect(def, sourceUser...) {
		return ActiveBuff{}, false
	}
	before := s.speeds()

	s.equipment()
	s.replaceAttackElement(def)
	id := uint32(0)
	kept := s.buffs[:0]
	refused := false
	for _, b := range s.buffs {
		if b.SkillID == def.SkillID && (b.SkillLevel == gemSkillLevel) == (def.SkillLevel == gemSkillLevel) {
			if def.SkillLevel <= 3 && def.SkillLevel < b.SkillLevel {
				refused = true
				kept = append(kept, b)
				continue
			}
			id = b.ID
			continue
		}
		kept = append(kept, b)
	}
	s.buffs = kept
	if refused {
		return ActiveBuff{}, false
	}
	if id == 0 {
		for _, b := range s.buffs {
			id = max(id, b.ID)
		}
		id++
	}
	b := newBuff(id, def, time.Now().Add(time.Duration(def.KeepSeconds)*time.Second), s.log)
	b.CasterMob = casterMob
	if len(sourceUser) == 1 {
		b.CasterUser = sourceUser[0]
	}
	s.buffs = append(s.buffs, b)
	if c := s.cast; c != nil {
		_, pending, known := s.knownSkill(c.number)
		if s.actionBlocked() || known && s.stats().Disabled[min(int(pending.Kind), 2)] > 0 {
			s.cast = nil
			s.log.Printf("GX Server World prepared cast interrupted by applied skill skill_id=%d", def.SkillID)
		}
	}
	s.syncVisibility()
	s.applyElement(b)
	s.send(buildBuffAdd(id, def.SkillID, def.SkillLevel), "0x050D skill applied")
	s.sendBuffStats("after applied skill")
	if after := s.speeds(); after != before {
		s.broadcastSpeed(after)
	}
	return b, true
}

// canAddEffect also guards item consumption, so a refused lower-level buff
// cannot consume a candy or start its cooldown.
func (s *Session) canAddEffect(def *db.SkillDef, sourceUser ...uint32) bool {
	self := s.char != nil && len(sourceUser) == 1 && sourceUser[0] == s.char.ID && def.TypeEffect == typeEffectBuff
	if s.invulnerable() && !self {
		return false
	}
	for _, b := range s.buffs {
		if b.SkillID == def.SkillID && (b.SkillLevel == gemSkillLevel) == (def.SkillLevel == gemSkillLevel) && def.SkillLevel <= 3 && def.SkillLevel < b.SkillLevel {
			return false
		}
	}
	return true
}

// buildBuffAdd (0x050D): instance id, skill id, level (the client reads
// nothing more).
func buildBuffAdd(id uint32, skillID uint16, level uint8) []byte {
	w := packet.New(packet.BuffAdd)
	w.U32(id)
	w.U16(skillID)
	w.U8(level)
	return w.MustFinish()
}

type playerSpeeds struct{ attack, move uint8 }

// blocks counts the action/movement blocks (details 64, 65, 66).
func (s *Session) blocks() (actionA, move, actionB int) {
	for _, b := range s.buffs {
		if b.Effect != typeEffectDebuff {
			continue
		}
		switch b.Detail {
		case detailStun:
			actionA++
		case detailStop:
			move++
		case detailSleep:
			actionB++
		}
	}
	return
}

// actionBlocked: no attacks or skills while stunned or asleep.
func (s *Session) actionBlocked() bool {
	a, _, b := s.blocks()
	return a > 0 || b > 0
}

// speeds is CUser::GetAttackSpeed / GetMoveSpeed: 255 while blocked.
func (s *Session) speeds() playerSpeeds {
	a, m, b := s.blocks()
	st := s.stats()
	sp := playerSpeeds{attack: 255, move: 255}
	if a == 0 && b == 0 {
		at := uint8(0)
		if wc, ok := equippedWeaponCombat(s.items); ok {
			at = wc.AttackTime
		}
		if s.cast == nil {
			sp.attack = attackspeed.Tier(at, st.AttackSpeed)
		}
		if m == 0 {
			sp.move = movespeed.Tier(st.MoveSpeed)
		}
	}
	return sp
}

// buildCharacterSpeed encodes 0x051C (character id, attack speed, move speed).
func buildCharacterSpeed(id uint32, sp playerSpeeds) []byte {
	w := packet.New(packet.CharacterDeathState)
	w.U32(id)
	w.U8(sp.attack)
	w.U8(sp.move)
	return w.MustFinish()
}

// syncPlayerSpeed publishes only server-derived state. Enter-view snapshots
// carry the same value, including equipment and restored consumable buffs.
func (s *Session) syncPlayerSpeed(force bool) {
	if s.char == nil {
		return
	}
	sp := s.speeds()
	if force || !s.speedKnown || sp != s.lastSpeeds {
		s.broadcastSpeed(sp)
	}
}

func (s *Session) broadcastSpeed(sp playerSpeeds) {
	if s.char == nil {
		return
	}
	moveChanged := s.speedKnown && s.lastSpeeds.move != sp.move
	s.lastSpeeds, s.speedKnown = sp, true
	// Do not bank faster movement across a tier change. A small existing credit
	// absorbs transport jitter; future credit is earned at the new server rate.
	if m := &s.movement; moveChanged && m.geometry != nil {
		m.credit = min(m.credit, .2)
		m.budgetAt = time.Now()
	}
	if p := s.player; p != nil {
		p.mu.Lock()
		p.speeds, p.speedKnown = sp, true
		p.mu.Unlock()
	}
	frame := buildCharacterSpeed(s.char.ID, sp)
	s.send(frame, "0x051C character speed")
	s.sendToViewers(frame, "0x051C character speed")
}

// removeEffects drops the matching applied skills with 0x050E and
// refreshes the stats and speeds.
func (s *Session) removeEffects(match func(*ActiveBuff) bool, why string) {
	before := s.speeds()
	kept := s.buffs[:0]
	var removed []ActiveBuff
	for i := range s.buffs {
		if match(&s.buffs[i]) {
			removed = append(removed, s.buffs[i])
			continue
		}
		kept = append(kept, s.buffs[i])
	}
	s.buffs = kept
	if len(removed) == 0 {
		return
	}
	for _, b := range removed {
		s.removeElement(b)
		w := packet.New(packet.BuffRemove)
		w.U32(b.ID)
		s.send(w.MustFinish(), "0x050E applied skill removed "+why)
	}
	s.syncVisibility()
	s.sendBuffStats(why)
	if after := s.speeds(); after != before {
		s.broadcastSpeed(after)
	}
}

// wake is CUser::RemApplySkillSleep: damage ends every sleep (detail 66).
func (s *Session) wake() {
	s.removeEffects(func(b *ActiveBuff) bool { return b.Effect == typeEffectDebuff && b.Detail == detailSleep }, "(woke)")
}

// applyMobDebuff leaves a mob skill (TypeEffect 3) on the player.
func (s *Session) applyMobDebuff(sk *db.SkillDef, mobGlobalID uint32) {
	if sk.TypeEffect != typeEffectDebuff || s.char == nil || s.char.CurrentHP == 0 {
		return
	}
	if b, ok := s.addEffect(sk, mobGlobalID); ok {
		s.log.Printf("GX Server World debuff applied skill_id=%d level=%d detail=%d keep=%d mob_global_id=%d",
			sk.SkillID, sk.SkillLevel, sk.TypeDetail, sk.KeepSeconds, mobGlobalID)
		_ = b
	}
}

// playerTimeDamage is CUser::TimeDamage (0x00486290) on the player.
func playerTimeDamage(def *db.SkillDef, st *CharacterStats, elapsed time.Duration) [3]int32 {
	cur := [3]uint32{st.CurrentHP, st.CurrentSP, st.CurrentMP}
	maxv := [3]uint32{st.MaxHP, st.MaxSP, st.MaxMP}
	return mobTimeDamage(def, cur, maxv, elapsed)
}

// tickDebuffs runs the damage of applied debuffs: detail 61 every 3 s and
// on expiry, detail 63 on expiry. Returns whether the player died.
func (s *Session) tickDebuffs(now time.Time) bool {
	if s.invulnerable() {
		return false
	}
	ch := s.char
	effects := append([]ActiveBuff(nil), s.buffs...)
	for i := range effects {
		b := &effects[i]
		if b.Effect != typeEffectDebuff || (b.Detail != detailTimeDamage && b.Detail != detailDeathSentence) {
			continue
		}
		def := db.MustDefs().Skill(b.SkillID, b.SkillLevel)
		if def == nil {
			continue
		}
		expiring := !b.EndsAt.After(now)
		var damage [3]int32
		switch {
		case b.Detail == detailTimeDamage && (expiring || now.Sub(b.prevTick) >= effectTick):
			st := s.stats()
			damage = playerTimeDamage(def, &st, now.Sub(b.prevTick))
			b.prevTick = now
			for j := range s.buffs {
				if s.buffs[j].ID == b.ID {
					s.buffs[j].prevTick = now
					break
				}
			}
		case b.Detail == detailDeathSentence && expiring:
			damage[0] = int32(min(s.stats().CurrentHP, 65535))
		default:
			continue
		}
		if damage[0] > 0 || damage[1] > 0 || damage[2] > 0 {
			s.breakStealth()
		}
		hp := uint32(max(damage[0], 0))
		sp, mp := uint32(max(damage[1], 0)), uint32(max(damage[2], 0))
		if sp > 0 || mp > 0 {
			s.spendDelta(mp, sp)
			if hp == 0 {
				s.persist()
			}
		}
		var killed bool
		if b.CasterUser != 0 {
			killed = s.takeUserDamage(b.CasterUser, hp)
		} else {
			killed = s.takeMobDamage(pendingDamage{MobGlobalID: b.CasterMob}, hp, nil)
		}
		w := packet.New(packet.SkillKeep)
		w.U32(ch.ID)
		w.U16(b.SkillID)
		w.U8(b.SkillLevel)
		w.U16(uint16(min(hp, 65535)))
		w.U16(uint16(min(max(damage[1], 0), 65535)))
		w.U16(uint16(min(max(damage[2], 0), 65535)))
		sendNear(ch.MapID, ch.PosX, ch.PosZ, CombatRadius, w.MustFinish(), "0x050F debuff damage")
		if killed {
			if b.CasterUser != 0 {
				s.afterUserDamage(b.CasterUser, true)
			} else {
				s.afterMobDamage(pendingDamage{MobGlobalID: b.CasterMob}, true)
			}
			s.dead, s.autoAttack, s.cast = true, false, nil
			s.clearBuffsOnDeath()
			return true
		}
		st := s.stats()
		s.send(buildCurrentHitpoints(&st), "0x0521 current hp after debuff damage")
	}
	return false
}

// blocksAttack is the defender's block check before CMob::AttackSuccess:
// detail 92 blocks everything; 85 blocks DefenceValue attacks of its
// category and then ends; 86 blocks with a DefenceValue% chance.
func (s *Session) blocksAttack(kind int) bool {
	for i := range s.buffs {
		b := &s.buffs[i]
		if b.Effect != typeEffectBuff {
			continue
		}
		switch b.Detail {
		case 92:
			return true
		case 85:
			if int(b.DefenceType) == kind && b.BlockLeft > 0 {
				b.BlockLeft--
				if b.BlockLeft == 0 {
					id := b.ID
					s.removeEffects(func(x *ActiveBuff) bool { return x.ID == id }, "(block used up)")
				}
				return true
			}
		case 86:
			if int(b.DefenceType) == kind && (b.BlockLeft >= 100 || rand.Uint32N(100) <= b.BlockLeft) {
				return true
			}
		}
	}
	return false
}

// deathExpLoss is CUser::Death's percent: 0 with an exp-protection
// ability (38) or the Red Phoenix charm (42), 1 with the Blue Dragon (40)
// or White Tiger (41) charm, otherwise 5.
func (s *Session) deathExpLoss() int {
	charm, protected := 0, false
	for _, b := range s.buffs {
		for _, a := range b.Abilities {
			switch a.Type {
			case 38:
				protected = protected || a.Value > 0
			case 40, 41, 42:
				if a.Value > 0 {
					charm = max(charm, int(a.Type)-39)
				}
			}
		}
	}
	switch {
	case protected || charm == 3:
		return 0
	case charm > 0:
		return 1
	}
	return deathExpPercent
}

// reflects is CUser::ChkMirror (0x0044e080) with a detail-83 buff covering
// the category (its AbilityValue for that category is non-zero):
// SuccessType 1 rolls SuccessValue%, 2 rolls LUC x 10 / SuccessValue and
// 3 WIS x 10 / SuccessValue (CUser +0x1234 / +0x122c). A mob resisting the
// melee mirror skill cannot be reflected; md is nil for a user attacker.
func (s *Session) reflects(kind int, md *db.MobDef) bool {
	var mirror, melee *db.SkillDef
	for _, b := range s.buffs {
		if b.Effect != typeEffectBuff || b.Detail != 83 {
			continue
		}
		d := db.MustDefs().Skill(b.SkillID, b.SkillLevel)
		if d == nil {
			continue
		}
		if d.AbilityValue[0] != 0 {
			melee = d
		}
		if kind < 3 && d.AbilityValue[kind] != 0 {
			mirror = d
		}
	}
	if mirror == nil {
		return false
	}
	st := s.stats()
	roll := rand.Int32N(100)
	value := int32(mirror.SuccessValue)
	reflected := false
	switch mirror.SuccessType {
	case 1:
		reflected = roll <= value
	case 2, 3:
		stat := int32(st.Luc)
		if mirror.SuccessType == 3 {
			stat = int32(st.Wis)
		}
		reflected = value != 0 && roll <= stat*10/value
	}
	if !reflected {
		return false
	}
	if melee != nil && md != nil {
		for _, id := range md.ResistSkill {
			if id == melee.SkillID {
				return false
			}
		}
	}
	return true
}

// reflectToMob sends a reflected hit back to the mob (0x051B: character,
// mob, HP/SP/MP).
func (s *Session) reflectToMob(mobGlobalID uint32, damage [3]uint16) {
	ch := s.char
	r, ok := DamageMobFrom(ch.MapID, mobGlobalID, uint32(damage[0]), ch.ID)
	if !ok {
		return
	}
	w := packet.New(0x051B)
	w.U32(ch.ID)
	w.U32(mobGlobalID)
	for _, v := range damage {
		w.U16(v)
	}
	sendNear(ch.MapID, ch.PosX, ch.PosZ, CombatRadius, w.MustFinish(), "0x051B reflected hit")
	s.resolveMobDamage(&r)
}
