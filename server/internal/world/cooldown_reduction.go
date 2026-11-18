package world

import (
	"math"
	"time"

	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/server/internal/db"
	"github.com/Spelunkern/Shaiya-GX/server/internal/packet"
)

const abilityCooldownCut = 19

// Cooldown reduction (ability 19): the value is a number of seconds taken off
// the reuse of spells.
//   - When the effect is applied, every spell already cooling down loses that
//     many seconds at once (a 60 second spell with 50 left and a 30 second
//     reduction has 20 left), and a spell whose remainder is covered is ready.
//   - While the effect lasts, a spell cast starts with its reuse reduced the
//     same way, so a reduction at least as long as a spell's reuse lets it be
//     cast again immediately.
// Potential skills keep their fixed reuse: they are not reduced, so a
// reduction cannot make one trigger again by itself.

// cooldownCut is the seconds currently taken off spell reuse.
func (s *Session) cooldownCut() uint16 {
	cut := s.stats().CooldownCut
	return uint16(min(max(cut, 0), 65535))
}

// sendSkillCooldown tells the client a skill's timer: remaining seconds and
// the full-bar total (0 keeps the one it has).
func (s *Session) sendSkillCooldown(skillID, remaining, total uint16) {
	w := packet.New(packet.Opcode(proto.GXSkillCooldown))
	w.U16(skillID)
	w.U16(remaining)
	w.U16(total)
	s.send(w.MustFinish(), "GX skill reuse")
}

// effectCooldownCut is the reduction a newly applied effect carries.
func effectCooldownCut(b *ActiveBuff) int32 {
	var cut int32
	for _, a := range b.Abilities {
		if a.Type == abilityCooldownCut && a.Value > 0 {
			cut += a.Value
		}
	}
	return cut
}

// reduceCooldowns takes seconds off every running spell reuse right now.
func (s *Session) reduceCooldowns(now time.Time, seconds int32) {
	if seconds <= 0 || s.char == nil || len(s.skillCooldowns) == 0 {
		return
	}
	reduced := false
	for id, at := range s.skillCooldowns {
		if def := s.learnedDef(id); isPotential(def) {
			continue
		}
		left := at.Sub(now)
		if left <= 0 {
			continue
		}
		left -= time.Duration(seconds) * time.Second
		if left <= 0 {
			delete(s.skillCooldowns, id)
			s.sendSkillCooldown(id, 0, 0)
		} else {
			s.skillCooldowns[id] = now.Add(left)
			s.sendSkillCooldown(id, uint16(min(math.Ceil(left.Seconds()), 65535)), 0)
		}
		reduced = true
	}
	if reduced {
		s.persistSkillDelays()
	}
}

// learnedDef is the definition of a learned skill at its learned level.
func (s *Session) learnedDef(id uint16) *db.SkillDef {
	for _, k := range s.skills {
		if k.SkillID == id {
			return db.MustDefs().Skill(k.SkillID, k.Level)
		}
	}
	return nil
}
