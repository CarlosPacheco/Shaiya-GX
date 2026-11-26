package world

import (
	"time"

	"github.com/Spelunkern/Shaiya-GX/internal/skilltiming"
	"github.com/Spelunkern/Shaiya-GX/server/internal/db"
	"github.com/Spelunkern/Shaiya-GX/server/internal/packet"
)

// reserveBasicAttack applies the absolute execution deadline used by
// CUser::UpdateAttack. Skill recovery may allow a release before a previous
// basic deadline, so a new reservation can only extend that deadline.
func (s *Session) reserveBasicAttack(now time.Time, tier uint8) {
	interval, ok := skilltiming.Interval(tier)
	if due := now.Add(interval); ok && due.After(s.nextBasicAttackAt) {
		s.nextBasicAttackAt = due
	}
}

func (s *Session) commitBasicAttack(now time.Time, tier uint8) {
	s.lastActionAt, s.lastCombatAt = now, now
	s.reserveBasicAttack(now, tier)
	s.commitAttackStop(now)
}

func (s *Session) tickAutoAttack() {
	if !s.autoAttack {
		return
	}
	swing := s.autoAttackSwing
	if s.autoAttackUser {
		swing = s.userAttackSwing
	}
	if !swing(s.autoAttackTarget) {
		s.autoAttack = false
	}
}

// actionInterval is the skill execution interval at the live attack index and weapon.
func (s *Session) actionInterval() (time.Duration, bool) {
	wc, have := equippedWeaponCombat(s.items)
	return skilltiming.IntervalFor(s.speeds().attack, wc.AttackTime, have)
}

// ownClip is the animation floor of the last skill; any later action stamp
// (basic attack, cast, item) makes it stale on its own.
func (s *Session) ownClip() time.Duration {
	if s.clipFloorAt.IsZero() || !s.clipFloorAt.Equal(s.lastActionAt) {
		return 0
	}
	return s.clipFloor
}

// actionRecovery is the next skill's interval, with a floor for the previous
// skill's own clip.
func (s *Session) actionRecovery() (time.Duration, bool) {
	interval, ok := s.actionInterval()
	return max(interval, s.ownClip()), ok
}

// combatActionReady checks skill recovery with the live equipment and buff tier.
func (s *Session) combatActionReady(now time.Time) bool {
	if s.dash != nil {
		return false
	}
	interval, ok := s.actionRecovery()
	return ok && (s.lastActionAt.IsZero() || now.Sub(s.lastActionAt) >= interval)
}

func (s *Session) skillActionReady(def *db.SkillDef, now time.Time) bool {
	if s.dash != nil {
		return false
	}
	if s.combatActionReady(now) {
		return true
	}
	_, allowed := s.actionInterval()
	return allowed && s.lastSkillAt.Equal(s.lastActionAt) && now.Sub(s.lastSkillAt) >= skilltiming.SpinContact && now.Sub(s.lastSkillAt) < skilltiming.SpinDuration && skilltiming.AreaCombo(s.lastSkillID, def.SkillID, def.TargetType, def.TypeEffect, def.ReadyTime)
}

type pendingSpinCombo struct {
	payload       []byte
	op            packet.Opcode
	due, previous time.Time
}

func (s *Session) queueSpinCombo(def *db.SkillDef, op packet.Opcode, payload []byte) bool {
	now := time.Now()
	if s.lastSkillAt.Equal(s.lastActionAt) && now.Before(s.lastSkillAt.Add(skilltiming.SpinContact)) && skilltiming.AreaCombo(s.lastSkillID, def.SkillID, def.TargetType, def.TypeEffect, def.ReadyTime) && s.skillReady(def.SkillID) {
		s.pendingCombo = &pendingSpinCombo{append([]byte(nil), payload...), op, s.lastSkillAt.Add(skilltiming.SpinContact), s.lastActionAt}
		return true
	}
	return false
}
func (s *Session) processSpinCombo() {
	q := s.pendingCombo
	if q == nil || time.Now().Before(q.due) {
		return
	}
	s.pendingCombo = nil
	if s.lastActionAt.Equal(q.previous) {
		s.handleUseSkill(q.op, q.payload)
	}
}

// pendingSkillUse is the classic one-slot selection: a request that reaches the
// server before the shared execution deadline is kept (latest wins) and runs the
// moment the interval allows, instead of being refused by a few milliseconds of
// network jitter. Own reuse, costs and validation are checked again at release.
type pendingSkillUse struct {
	payload []byte
	op      packet.Opcode
	expires time.Time
}

func (s *Session) holdSkillUse(def *db.SkillDef, op packet.Opcode, payload []byte, now time.Time) bool {
	if payload == nil || s.dash != nil || !s.skillReady(def.SkillID) || s.lastActionAt.IsZero() {
		return false
	}
	interval, ok := s.actionRecovery()
	if !ok {
		return false
	}
	s.pendingSkill = &pendingSkillUse{append([]byte(nil), payload...), op, s.lastActionAt.Add(interval + time.Second)}
	return true
}

func (s *Session) processPendingSkill() {
	q := s.pendingSkill
	if q == nil {
		return
	}
	now := time.Now()
	if now.After(q.expires) || s.dead || s.cast != nil {
		s.pendingSkill = nil
		return
	}
	_, def, ok := s.knownSkill(q.payload[2])
	if !ok {
		s.pendingSkill = nil
		return
	}
	if !s.skillActionReady(def, now) {
		return
	}
	s.pendingSkill = nil
	s.handleUseSkill(q.op, q.payload)
}

// minWake keeps a blocked pending action from spinning the loop.
const minWake = 5 * time.Millisecond

// nextWake is how long the session loop may sleep: the usual tick, shortened to
// the moment a held skill, a spin combo or the next auto-attack swing becomes
// due, so those run at their exact deadline instead of the next 100 ms tick.
func (s *Session) nextWake() time.Duration {
	wake, now := loopTick, time.Now()
	consider := func(due time.Time) {
		wake = min(wake, max(due.Sub(now), minWake))
	}
	if s.char != nil && !s.lastActionAt.IsZero() && s.pendingSkill != nil {
		if interval, ok := s.actionRecovery(); ok {
			consider(s.lastActionAt.Add(interval))
		}
	}
	if s.autoAttack && !s.nextBasicAttackAt.IsZero() {
		consider(s.nextBasicAttackAt)
	}
	if s.pendingCombo != nil {
		consider(s.pendingCombo.due)
	}
	return wake
}
