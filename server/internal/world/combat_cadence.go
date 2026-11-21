package world

import (
	"github.com/Spelunkern/Shaiya-GX/internal/skilltiming"
	"github.com/Spelunkern/Shaiya-GX/server/internal/db"
	"github.com/Spelunkern/Shaiya-GX/server/internal/packet"
	"time"
)

// Both basic attack routes and skills recover from lastActionAt. Use the live
// equipment/buff tier so haste, slow and gear changes affect recovery immediately.
func (s *Session) combatActionReady(now time.Time) bool {
	if s.dash != nil {
		return false
	}
	interval, ok := skilltiming.Interval(s.speeds().attack)
	return ok && (s.lastActionAt.IsZero() || now.Sub(s.lastActionAt) >= interval)
}

func (s *Session) skillActionReady(def *db.SkillDef, now time.Time) bool {
	if s.dash != nil {
		return false
	}
	if s.combatActionReady(now) {
		return true
	}
	_, allowed := skilltiming.Interval(s.speeds().attack)
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
	interval, ok := skilltiming.Interval(s.speeds().attack)
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
