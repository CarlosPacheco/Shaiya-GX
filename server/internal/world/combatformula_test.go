package world

import (
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/skillchance"
	"github.com/Spelunkern/Shaiya-GX/server/internal/db"
)

func TestCombatDEXTiersAndCritical(t *testing.T) {
	for _, tc := range []struct {
		dex  uint32
		want int64
	}{{0, 0}, {1, 10}, {499, 4990}, {500, 5000}, {501, 5005}, {800, 6500}, {801, 6501}, {4300, fullChance}} {
		if got := dexChance(tc.dex, 500, 800); got != tc.want {
			t.Errorf("DEX %d = %d, want %d", tc.dex, got, tc.want)
		}
	}
	if got := dexChance(701, 400, 700); got != 5501 {
		t.Errorf("custom thresholds = %d", got)
	}
	var s CharacterStats
	setAttack(&s, EquipmentStats{})
	if s.Critical != 0 || rollCombatChance(s.Critical) {
		t.Fatal("no luck or bonuses must yield no criticals")
	}
	s.Luc = 475
	setAttack(&s, EquipmentStats{})
	if s.Critical != 9500 {
		t.Fatalf("475 LUC = %d", s.Critical)
	}
	s.Luc = 500
	setAttack(&s, EquipmentStats{})
	if s.Critical != fullChance || !rollCombatChance(s.Critical) {
		t.Fatal("500 LUC must always crit")
	}
}

func TestCombatStatsSharedPhysicalGuardAndStraightMagic(t *testing.T) {
	stats := CharacterStats{Level: 60, Str: 100, Dex: 500, Int: 100, Wis: 500, Luc: 100, Rec: 50}
	e := EquipmentStats{ItemAttack: 200, Variance: 20, Defense: 40, Resistance: 80}
	e.Mods[attackMelee] = CategoryMods{Defense: 10, Evasion: 15}
	e.Mods[attackRanged] = CategoryMods{Defense: 30, Accuracy: 20}
	e.Mods[attackMagic] = CategoryMods{Defense: 10, Accuracy: 5}
	setAttack(&stats, e)
	if stats.Attack != ([3]int32{430, 480, 430}) || stats.MinAttack != 430 || stats.MaxAttack != 450 {
		t.Errorf("attacks and inclusive range: %v, %d–%d", stats.Attack, stats.MinAttack, stats.MaxAttack)
	}
	if stats.Guard != ([3]int32{100, 130, 590}) || stats.Defense != 100 || stats.Resistance != 590 {
		t.Errorf("derived defenses: %v, %d, %d", stats.Guard, stats.Defense, stats.Resistance)
	}
	if stats.Accuracy != ([3]int64{5000, 7000, 1500}) ||
		stats.Evasion != ([3]int64{6500, 5000, 1000}) || stats.Critical != 2000 {
		t.Errorf("chances: accuracy %v, evasion %v, critical %d", stats.Accuracy, stats.Evasion, stats.Critical)
	}
	e.KCGuardPct = [2]int32{25, 0}
	setAttack(&stats, e)
	bonus := int32(kcPercent(100, 25))
	if stats.Guard[attackMelee] != 100+bonus || stats.Guard[attackRanged] != 130+bonus {
		t.Errorf("common defense from blessing: %v", stats.Guard)
	}
	for _, kind := range []uint8{18, 20, 21, 22, 29, 30, 31} {
		if skillchance.Modifier(kind, 500) != 100 || skillchance.Modifier(kind, -500) != -100 ||
			skillchance.Modifier(kind, 1) != 1 {
			t.Errorf("skill chance amount for kind %d", kind)
		}
	}
	if skillchance.Modifier(23, 500) != 500 {
		t.Fatal("flat attack is not a percentage ability")
	}
}

func TestCombatPvELevelAdjustment(t *testing.T) {
	s := CharacterStats{
		Level: 50, Attack: [3]int32{1000, 1000, 1000},
		Evasion: [3]int64{3000, 3000, 3000},
	}
	m := mobCombat{Evasion: [3]int64{2000, 2000, 2000}}
	attack := db.MobAttack{Attack: 1000}
	for _, tc := range []struct {
		level                          int32
		hit, dodge                     int64
		outgoingDamage, incomingDamage int32
	}{
		{49, 8000, 3000, 1000, 1000}, {50, 8000, 3000, 1000, 1000},
		{54, 6400, 2400, 800, 1200}, {60, 4000, 1500, 500, 1500},
		{80, 4000, 1500, 500, 1500},
	} {
		m.Level = tc.level
		for _, kind := range []int{attackMelee, attackRanged, attackMagic} {
			if hit, dodge := playerHitChance(&s, &m, kind), playerDodgeChance(&s, &m, kind); hit != tc.hit || dodge != tc.dodge {
				t.Errorf("level %d kind %d: hit/dodge = %d/%d", tc.level, kind, hit, dodge)
			}
			outgoing, _ := playerDamage(&s, &m, kind)
			incoming, _ := mobHitDamage(&m, &attack, &s, kind, 0).weaponHit(0, 1)
			if outgoing != tc.outgoingDamage || incoming != tc.incomingDamage {
				t.Errorf("level %d kind %d: damage = %d/%d", tc.level, kind, outgoing, incoming)
			}
		}
	}
}

func TestCombatDirectSkillsAndCatalogRoll(t *testing.T) {
	attacker := CharacterStats{Level: 60, AttackAttr: 1, Attack: [3]int32{1200, 1200, 1200}}
	defender := CharacterStats{Level: 1, DefenseAttr: 4, Guard: [3]int32{500, 500, 500}}
	skill := db.SkillDef{DamageType: 1, DamageHP: 200, SuccessType: 1, SuccessValue: 100}
	for kind := range 3 {
		skill.Kind = uint8(kind)
		if damage, crit := userSkillDamage(&attacker, &defender, &skill, damagePools{}); damage[0] != 1260 || crit {
			t.Fatalf("normal skill kind %d = %v, crit %t", kind, damage, crit)
		}
		attacker.Critical = fullChance
		if damage, crit := userSkillDamage(&attacker, &defender, &skill, damagePools{}); damage[0] != 1890 || !crit {
			t.Fatalf("critical skill kind %d = %v, crit %t", kind, damage, crit)
		}
		attacker.Critical = 0
		defender.Evasion[kind] = fullChance
		if userSkillHits(&attacker, &defender, &skill) {
			t.Fatal("an authored guaranteed roll still respects evasion")
		}
	}
	m := mobCombat{Level: 64}
	for _, tc := range []struct {
		damageType uint8
		want       [3]int32
	}{{0, [3]int32{80, 16, 8}}, {4, [3]int32{800, 80, 16}}, {5, [3]int32{400, 40, 8}}} {
		fixed := db.SkillDef{DamageType: tc.damageType, DamageHP: 100, DamageSP: 20, DamageMP: 10}
		got, crit := playerSkillDamage(&attacker, &m, &fixed, damagePools{500, 250, 100}, damagePools{1000, 500, 200})
		if got != tc.want || crit {
			t.Errorf("skill mode %d = %v, want %v, crit %t", tc.damageType, got, tc.want, crit)
		}
	}
}

// Each LUC point adds 0.5 ranged attack and the target's LUC takes 0.01 percentage
// points off the attacker's critical chance.
func TestLuckRangedAttackAndCriticalReduction(t *testing.T) {
	var s CharacterStats
	s.Str, s.Dex, s.Luc = 100, 50, 200
	setAttack(&s, EquipmentStats{})
	if melee, ranged := s.Attack[attackMelee], s.Attack[attackRanged]; melee != 140 || ranged != 240 {
		t.Fatalf("melee %d ranged %d", melee, ranged)
	}
	if s.Critical != 200*20 {
		t.Fatalf("critical %d", s.Critical)
	}
	for _, c := range []struct{ crit, luc, want int64 }{{4000, 0, 4000}, {4000, 500, 3500}, {100, 500, 0}, {20000, 0, 10000}} {
		if got := criticalChance(c.crit, c.luc); got != c.want {
			t.Errorf("criticalChance(%d,%d)=%d want %d", c.crit, c.luc, got, c.want)
		}
	}
	// A target with enough LUC can never be critically hit.
	h := hitDamage{attack: 1000, critical: 500, targetLuc: 500, defenseAttr: 0, levelPercent: 100}
	for i := 0; i < 200; i++ {
		if _, crit := h.weaponHit(0, 1); crit {
			t.Fatal("critical despite the target's LUC")
		}
	}
}
