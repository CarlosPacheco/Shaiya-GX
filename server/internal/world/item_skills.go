package world

import (
	"github.com/Spelunkern/Shaiya-GX/server/internal/db"
	"github.com/Spelunkern/Shaiya-GX/server/internal/packet"
	"time"
)

// All consumable Range/AttackTime buff references use the same native
// UseItemSkill -> BuffUser presentation, including commercial item skills.
func itemBuffSkill(d *db.ItemDef) *db.SkillDef {
	if d.Range == 0 || d.AttackTime == 0 {
		return nil
	}
	sk := db.MustDefs().EffectSkill(uint16(d.Range), d.AttackTime)
	if sk == nil || sk.TypeEffect != typeEffectBuff {
		return nil
	}
	return sk
}
func (s *Session) announceItemSkill(sk *db.SkillDef) {
	now := time.Now()
	s.lastActionAt = now
	s.lastSkillID = 0
	s.pendingCombo = nil
	s.autoAttack = false
	s.commitAttackStop(now)
	s.broadcastNear(buildSkillUseResult(packet.SkillUseNoTarget, 0, s.char.ID, s.char.ID, sk.SkillID, sk.SkillLevel, 0, 0, 0), "item skill release", true)
}

// Native ItemUse special 50: only the two arena maps, after half a level.
func (s *Session) canUseExpReset() bool {
	if s.char.MapID != 18 && s.char.MapID != 30 {
		return false
	}
	prev, next := expBounds(s.char)
	return next > prev && uint64(s.char.Exp) >= (uint64(prev)+uint64(next))/2
}
