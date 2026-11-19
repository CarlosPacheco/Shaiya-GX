package world

import (
	"time"

	"github.com/Spelunkern/Shaiya-GX/server/internal/db"
)

const (
	detailDetection = 93          // buff: reveals the enemies hiding around the caster
	detectionTick   = time.Second // how often the area is swept
)

// RevealStealth asks a user's session to end its stealth.
func (p *ConnectedPlayer) RevealStealth() {
	p.mailMu.Lock()
	p.reveal = true
	p.mailMu.Unlock()
}

func (p *ConnectedPlayer) takeReveal() bool {
	p.mailMu.Lock()
	defer p.mailMu.Unlock()
	r := p.reveal
	p.reveal = false
	return r
}

// tickReveal ends the stealth of a user an enemy's Detection has found
// (RemApplySkillHide): it is not hidden again until it casts Stealth again.
func (s *Session) tickReveal() {
	if s.player != nil && s.player.takeReveal() {
		s.breakStealth()
	}
}

// tickDetection is CUser::ChkFindHide, run by ChkKeepSkill for a Detection
// buff (TypeDetail 93): on every sweep each living enemy user within
// ApplyRange of the caster loses its stealth. The area is the caster's current
// position, so it follows the caster; allies, Game Masters who are concealed
// and users that are not hidden are untouched. The first sweep runs when the
// buff is applied.
func (s *Session) tickDetection(now time.Time) {
	if s.char == nil || s.dead {
		return
	}
	var radius float32
	for i := range s.buffs {
		b := &s.buffs[i]
		if b.Effect != typeEffectBuff || b.Detail != detailDetection || b.nextTickAt.IsZero() || now.Before(b.nextTickAt) {
			continue
		}
		b.nextTickAt = now.Add(detectionTick)
		if def := db.MustDefs().EffectSkill(b.SkillID, b.SkillLevel); def != nil {
			radius = max(radius, float32(def.ApplyRange))
		}
	}
	if radius == 0 {
		return
	}
	ch := s.char
	for _, p := range s.hostileUsersWithin(ch.PosX, ch.PosZ, radius) {
		if p.hidden.Load() && p.GMVisible() {
			p.RevealStealth()
		}
	}
}
