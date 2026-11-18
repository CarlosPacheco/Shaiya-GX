package world

import (
	"testing"

	"github.com/Spelunkern/Shaiya-GX/server/internal/db"
)

// Sacrifice (TypeEffect 2, TypeDetail 43): the caster pays its whole SP/MP pool
// and the target recovers the caster's maximum SP (level 1), SP and MP (2) or
// HP (level 3).
func TestSacrificeTransfersTheCastersMaximumPool(t *testing.T) {
	for level := uint8(1); level <= 3; level++ {
		t.Run(string('0'+level), func(t *testing.T) { sacrificeLevel(t, level) })
	}
}

func sacrificeLevel(t *testing.T, level uint8) {
	s, _ := potentialSession(t)
	def := db.MustDefs().Skill(106, level)
	if def == nil || def.TypeDetail != detailSacrificeHeal {
		t.Fatalf("Sacrifice %d missing: %+v", level, def)
	}
	k := db.CharacterSkill{SkillID: 106, Level: level, Number: 1}
	s.skills = []db.CharacterSkill{k}
	st := s.stats()
	s.char.CurrentHP, s.char.CurrentSP, s.char.CurrentMP = st.MaxHP/2, st.MaxSP/2, st.MaxMP/2
	before := s.stats()
	s.friendlySkill(k, def, s.char.ID, before)
	after := s.stats()
	switch level {
	case 1:
		if after.CurrentSP != st.MaxSP || after.CurrentMP != before.CurrentMP || after.CurrentHP != before.CurrentHP {
			t.Fatalf("level 1: SP %d MP %d HP %d", after.CurrentSP, after.CurrentMP, after.CurrentHP)
		}
	case 2:
		if after.CurrentSP != st.MaxSP || after.CurrentMP != st.MaxMP || after.CurrentHP != before.CurrentHP {
			t.Fatalf("level 2: SP %d MP %d HP %d", after.CurrentSP, after.CurrentMP, after.CurrentHP)
		}
	case 3:
		if after.CurrentHP != st.MaxHP || after.CurrentSP != 0 || after.CurrentMP != 0 {
			t.Fatalf("level 3: HP %d SP %d MP %d", after.CurrentHP, after.CurrentSP, after.CurrentMP)
		}
	}
	if s.skillReady(106) {
		t.Fatalf("level %d did not start its reuse", level)
	}
}
