package app

import (
	"strings"
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/data/sdata"
)

func TestSkillTooltipTextFollowsEffectiveCatalog(t *testing.T) {
	for _, tc := range []struct {
		label string
		id    uint16
		npc   bool
		def   sdata.Skill
		name  string
		want  []string
	}{
		{"hostile area", 40, false, sdata.Skill{Name: "Ground Shock Lv2", Effect: 3, Detail: 62, Target: 6, ApplyRange: 4, Description: "Slows enemies within 8 meters."}, "", []string{"within 4 m"}},
		{"party healing", 325, false, sdata.Skill{Name: "Party Heal", Effect: 2, Detail: 41, Target: 5, ApplyRange: 15, Heal: [3]int16{491}}, "", []string{"491 HP plus twice your WIS", "you and party members within 15 m"}},
		{"periodic MP", 279, false, sdata.Skill{Name: "Continuous Mana Recovery L...", Effect: 4, Detail: 81, KeepTime: 60, TimeHeal: [3]int16{0, 0, 28}}, "Continuous Mana Recovery Lv1", []string{"28 MP every 3 seconds for 60 seconds"}},
		{"slow ranks", 33, false, sdata.Skill{Name: "Slow", Effect: 3, Detail: 62, Target: 3, KeepTime: 9, AbilityType: [3]uint8{17, 16}, AbilityValue: [3]int16{2, 0}, SuccessType: 1, SuccessValue: 80}, "", []string{"movement speed by 2 tiers and attack speed by 1 tier", "9 seconds", "80%"}},
		{"fire thorn", 47, false, sdata.Skill{Name: "Fire Thorn", Effect: 4, Detail: 84, ApplyRange: 2, KeepTime: 300, Damage: [3]uint16{8}}, "", []string{"8 damage every 2 seconds", "within 6 m"}},
		{"player poison and prerequisite", 79, false, sdata.Skill{Name: "Venom Crash", Effect: 3, Detail: 61, KeepTime: 15, TimeDamageType: 5, TimeDamage: [3]uint16{2}, PreviousSkill: 135, Description: "Chain skill that requires Drag Down; Poison."}, "", []string{"2% of the target's current HP every 3 seconds for 15 seconds", "Drag Down is cooling down."}},
		{"health assistant", 74, false, sdata.Skill{Name: "Health Assistant Lv2", Effect: 4, Detail: 87, KeepTime: 60}, "", []string{"spends MP to absorb incoming hit damage"}},
		{"NPC self heal", 332, true, sdata.Skill{Name: "Divine Grace", Effect: 2, Detail: 41, Target: 6, ApplyRange: 40, Heal: [3]int16{20000}}, "", []string{"Restores 20000 HP to the caster."}},
		{"NPC unused periodic mode", 259, true, sdata.Skill{Name: "Hell Flame", Description: "Continuous fire damage for 45 minutes", Effect: 3, Detail: 61, DamageType: 1, Attack: 3, Damage: [3]uint16{1500}, TimeDamage: [3]uint16{750}, TimeDamageType: 1}, "", []string{"Deals magic damage with a bonus of 1500."}},
		{"NPC resource poison", 10, true, sdata.Skill{Name: "Illness", Effect: 3, Detail: 61, KeepTime: 120, TimeDamageType: 5, TimeDamage: [3]uint16{0, 2}}, "", []string{"2% of the target's current SP every 3 seconds for 120 seconds"}},
		{"NPC silence", 3, true, sdata.Skill{Name: "Silent", Effect: 3, Detail: 67, KeepTime: 20, AbilityType: [3]uint8{34}, AbilityValue: [3]int16{1}}, "Silence", []string{"20 seconds", "blocks magic skills"}},
		{"NPC area stun", 172, true, sdata.Skill{Name: "Ice Stun", Effect: 3, Detail: 64, Target: 7, ApplyRange: 35, KeepTime: 15, DamageType: 4, Damage: [3]uint16{25}}, "", []string{"25% of the target's maximum HP", "Stuns targets for 15 seconds", "35 m around the target"}},
		{"NPC death sentence", 173, true, sdata.Skill{Name: "Death Breath", Effect: 3, Detail: 63, Target: 7, KeepTime: 20, DamageType: 5, Damage: [3]uint16{20}}, "", []string{"20% of the target's current HP", "remaining HP after 20 seconds unless removed"}},
	} {
		t.Run(tc.label, func(t *testing.T) {
			key := sdata.SkillKey{ID: tc.id, Level: 1}
			if tc.id == 74 && !tc.npc {
				key.Level = 2
			}
			defs := map[sdata.SkillKey]sdata.Skill{key: tc.def, {ID: 135, Level: 1}: {Name: "Drag Down Lv1"}}
			refreshSkillTooltips(defs, tc.npc)
			got := defs[key]
			for _, want := range tc.want {
				if !strings.Contains(got.Description, want) {
					t.Errorf("tooltip = %q, missing %q", got.Description, want)
				}
			}
			if tc.name != "" && got.Name != tc.name {
				t.Errorf("name = %q, want %q", got.Name, tc.name)
			}
			resolved := got
			resolved.Name, resolved.Description = tc.def.Name, tc.def.Description
			if resolved != tc.def {
				t.Fatalf("tooltip changed non-text data: %+v", resolved)
			}
			refreshSkillTooltips(defs, tc.npc)
			if defs[key] != got {
				t.Fatalf("reapplying text changed the tooltip: %q", defs[key].Description)
			}
		})
	}
}

func TestUnmodeledNPCTooltipRetainsCatalogDescription(t *testing.T) {
	def := sdata.Skill{Name: "Porcine Party", Description: "Turns enemies into pigs.", Effect: 3, Detail: 90}
	refreshNPCTooltip(100, &def)
	if def.Description != "Turns enemies into pigs." {
		t.Fatalf("unmodeled skill lost its description: %q", def.Description)
	}
}
