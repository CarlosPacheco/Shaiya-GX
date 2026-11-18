package world

import (
	"time"

	"github.com/Spelunkern/Shaiya-GX/internal/potential"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/skilltiming"
	"github.com/Spelunkern/Shaiya-GX/server/internal/db"
	"github.com/Spelunkern/Shaiya-GX/server/internal/packet"
)

func isPotential(def *db.SkillDef) bool {
	return def != nil && potential.Is(def.TypeEffect, def.TypeDetail)
}

// tickPotentials runs the Potential skills (TypeEffect 0, TypeDetail 6): a
// learned Potential whose LimitHP percent is above the character's current HP
// percent applies itself as an ordinary buff for its KeepTime and starts its
// reuse timer. It cannot be cast, so the reuse only ever starts here. Its
// abilities are the ones in the definition; Potential Motive's ability 19
// (skip the preparation of short-reuse spells) is intentionally not modeled.
func (s *Session) tickPotentials(now time.Time) {
	if s.char == nil || s.dead || len(s.skills) == 0 {
		return
	}
	var st CharacterStats
	haveStats := false
	for _, k := range s.skills {
		def := db.MustDefs().Skill(k.SkillID, k.Level)
		if !isPotential(def) || def.LimitHP == 0 || !s.skillReady(k.SkillID) || s.hasBuff(k.SkillID) {
			continue
		}
		if !haveStats {
			st, haveStats = s.stats(), true
		}
		// Strictly below the margin: HP / MaxHP < LimitHP percent.
		if st.MaxHP == 0 || st.CurrentHP == 0 || uint64(st.CurrentHP)*100 >= uint64(def.LimitHP)*uint64(st.MaxHP) {
			continue
		}
		if _, ok := s.addEffect(def, 0, s.char.ID); !ok {
			continue // refused: no reuse is spent
		}
		reuse := skilltiming.ReuseSeconds(def.ResetTimeSeconds)
		s.skillCooldowns[k.SkillID] = now.Add(time.Duration(reuse) * time.Second)
		k.Delay = int16(min(reuse, 32767))
		if err := db.SaveCharacterSkill(s.char.ID, k); err != nil {
			s.log.Printf("GX Server World failed to save potential cooldown: %v", err)
		}
		w := packet.New(packet.Opcode(proto.GXSkillCooldown))
		w.U16(k.SkillID)
		w.U16(uint16(min(reuse, 65535)))
		s.send(w.MustFinish(), "GX potential reuse started")
		s.log.Printf("GX Server World potential triggered skill_id=%d level=%d hp=%d/%d", k.SkillID, k.Level, st.CurrentHP, st.MaxHP)
	}
}

// hasBuff reports whether this skill is already applied.
func (s *Session) hasBuff(skillID uint16) bool {
	for _, b := range s.buffs {
		if b.SkillID == skillID {
			return true
		}
	}
	return false
}
