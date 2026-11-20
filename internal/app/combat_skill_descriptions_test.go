package app

import (
	"strings"
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/data/sdata"
)

func TestCombatSkillDescriptionsUseEffectivePercentages(t *testing.T) {
	defs := map[sdata.SkillKey]sdata.Skill{
		{ID: 26, Level: 2}:  {Effect: 0, Detail: 6, LimitHP: 15, AbilityType: [3]uint8{28, 18, 20}, AbilityValue: [3]int16{50, 120, 15}},
		{ID: 136, Level: 3}: {KeepTime: 60, AbilityType: [3]uint8{20, 21}, AbilityValue: [3]int16{500, 500}},
		{ID: 189, Level: 2}: {KeepTime: 20, AbilityType: [3]uint8{29, 30}, AbilityValue: [3]int16{180, 180}},
	}
	normalizePotentials(defs)
	normalizeCombatSkillDescriptions(defs)
	for _, tc := range []struct {
		key  sdata.SkillKey
		want string
	}{
		{sdata.SkillKey{ID: 26, Level: 2}, "100 percentage points of critical chance"},
		{sdata.SkillKey{ID: 136, Level: 3}, "100 percentage points of melee accuracy and 100 of ranged accuracy for 60 seconds"},
		{sdata.SkillKey{ID: 189, Level: 2}, "100 percentage points of melee evasion and 100 of ranged evasion for 20 seconds"},
	} {
		if got := defs[tc.key].Description; !strings.Contains(got, tc.want) {
			t.Errorf("%v description = %q, want %q", tc.key, got, tc.want)
		}
	}
}
