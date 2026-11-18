package world

import (
	"testing"

	"github.com/Spelunkern/Shaiya-GX/server/internal/db"
)

// Health Assistant (buff 4/87): level 1 spends SP against a hit, level 2 MP,
// level 3 SP and then MP, point for point; HP takes what is left.
func TestHealthAssistantAbsorbsHitsWithSPAndMP(t *testing.T) {
	cases := []struct {
		level          uint8
		sp, mp, damage uint32
		hp, sp2, mp2   uint32 // damage left for HP, SP and MP left
	}{
		{1, 100, 500, 30, 0, 70, 500},
		{1, 20, 500, 30, 10, 0, 500}, // SP runs out: HP takes the rest
		{2, 500, 100, 30, 0, 500, 70},
		{2, 500, 20, 30, 10, 500, 0},
		{3, 20, 100, 50, 0, 0, 70}, // SP first, then MP
		{3, 20, 10, 50, 20, 0, 0},  // both run out
	}
	for i, c := range cases {
		t.Run(string(rune('a'+i)), func(t *testing.T) { healthAssistCase(t, c) })
	}
	t.Run("none", func(t *testing.T) {
		s, _ := potentialSession(t)
		if s.absorbHealthAssist(40) != 40 {
			t.Fatal("without the buff HP takes everything")
		}
	})
}

func healthAssistCase(t *testing.T, c struct {
	level          uint8
	sp, mp, damage uint32
	hp, sp2, mp2   uint32
}) {
	{
		s, _ := potentialSession(t)
		def := db.MustDefs().Skill(74, c.level)
		if def == nil || def.TypeDetail != detailHealthAssist {
			t.Fatalf("Health Assistant %d definition: %+v", c.level, def)
		}
		if _, ok := s.addEffect(def, 0, s.char.ID); !ok {
			t.Fatal("the buff was not applied")
		}
		s.char.CurrentSP, s.char.CurrentMP = c.sp, c.mp
		st := s.stats()
		s.char.CurrentSP, s.char.CurrentMP = min(c.sp, st.MaxSP), min(c.mp, st.MaxMP)
		before := s.stats()
		if c.sp > before.MaxSP || c.mp > before.MaxMP {
			t.Fatalf("fixture pools too small: %d/%d", before.MaxSP, before.MaxMP)
		}
		if rest := s.absorbHealthAssist(c.damage); rest != c.hp {
			t.Fatalf("level %d: HP damage %d, want %d", c.level, rest, c.hp)
		}
		after := s.stats()
		if want := min(c.sp, before.MaxSP) - (c.sp - c.sp2); after.CurrentSP != want && c.sp <= before.MaxSP {
			t.Fatalf("level %d: SP %d, want %d", c.level, after.CurrentSP, want)
		}
		if c.mp <= before.MaxMP && after.CurrentMP != c.mp2 {
			t.Fatalf("level %d: MP %d, want %d", c.level, after.CurrentMP, c.mp2)
		}
	}
}
