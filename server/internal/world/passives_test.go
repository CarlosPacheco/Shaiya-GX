package world

import (
	"testing"

	"github.com/Spelunkern/Shaiya-GX/server/internal/db"
)

// Weapon-power and shield-defence passives feed the final attack and defense
// that 0x0526 reports (CUser::AddSkillPassive).
func TestWeaponAndShieldPassivesRaiseFinalStats(t *testing.T) {
	e := EquipmentStats{WeaponType: 1, ShieldType: 19}
	applyPassives(&e, []db.CharacterSkill{{SkillID: 5, Level: 3}, {SkillID: 303, Level: 3}})
	var s CharacterStats
	setAttack(&s, e)
	if s.MinAttack != 20 || s.Defense != 89 {
		t.Fatalf("attack %d defense %d, want 20 and 89", s.MinAttack, s.Defense)
	}
	// A different weapon type gets nothing from the sword mastery.
	e = EquipmentStats{WeaponType: 2}
	applyPassives(&e, []db.CharacterSkill{{SkillID: 5, Level: 3}})
	if e.ItemAttack != 0 {
		t.Fatal("mastery applied to the wrong weapon type")
	}
}
