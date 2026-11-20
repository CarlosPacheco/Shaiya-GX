package world

import (
	"math"
	"math/rand/v2"

	"github.com/Spelunkern/Shaiya-GX/internal/element"
	"github.com/Spelunkern/Shaiya-GX/server/internal/config"
	"github.com/Spelunkern/Shaiya-GX/server/internal/db"
)

const (
	attackMelee = iota
	attackRanged
	attackMagic
)

const (
	maxDamage  = 30000
	fullChance = 10000 // Hundredths of one percentage point.
)

var (
	combatDEXFirst  uint32 = config.DefaultCombatDEXFirst
	combatDEXSecond uint32 = config.DefaultCombatDEXSecond
)

func dexChance(dex, first, second uint32) int64 {
	return int64(min(dex, first))*10 +
		int64(min(dex, second)-min(dex, first))*5 +
		int64(dex-min(dex, second))
}

func setCombatChances(accuracy, evasion *[3]int64, dex, wis uint32, mods [3]CategoryMods) {
	physical := dexChance(dex, combatDEXFirst, combatDEXSecond)
	for kind, base := range [3]int64{physical, physical, int64(wis) * 2} {
		accuracy[kind] = base + int64(mods[kind].Accuracy)*100
		evasion[kind] = base + int64(mods[kind].Evasion)*100
	}
}

func boundedCombatStat(value int64) int32 {
	return int32(min(max(value, 0), math.MaxInt32))
}

// setAttack uses the same attribute scaling for melee and ranged. Category
// abilities still apply to their own attack, while melee defense is shared.
func setAttack(s *CharacterStats, e EquipmentStats) {
	physical := (int64(s.Str)*13 + int64(s.Dex)*2) / 10
	magic := (int64(s.Int)*13 + int64(s.Wis)*2) / 10
	for kind, base := range [3]int64{physical, physical, magic} {
		s.Attack[kind] = boundedCombatStat(base + int64(e.ItemAttack) + int64(e.Mods[kind].Attack))
	}
	guard := int64(s.Rec) + int64(e.Defense) + int64(e.Mods[attackMelee].Defense)
	guard += int64(kcPercent(uint32(boundedCombatStat(guard)), e.KCGuardPct[attackMelee]))
	rangedGuard := guard + int64(e.Mods[attackRanged].Defense)
	rangedGuard += int64(kcPercent(uint32(boundedCombatStat(rangedGuard)), e.KCGuardPct[attackRanged]))
	s.Guard = [3]int32{
		boundedCombatStat(guard),
		boundedCombatStat(rangedGuard),
		boundedCombatStat(int64(s.Wis) + int64(e.Resistance) + int64(e.Mods[attackMagic].Defense)),
	}
	setCombatChances(&s.Accuracy, &s.Evasion, s.Dex, s.Wis, e.Mods)
	s.Critical = int64(s.Luc)*20 + int64(e.Critical)*100
	// Each LUC point also gives half a point of ranged (archery) attack.
	s.Attack[attackRanged] = boundedCombatStat((int64(s.Str)*13+int64(s.Dex)*2+int64(s.Luc)*5)/10 + int64(e.ItemAttack) + int64(e.Mods[attackRanged].Attack))
	s.Variance, s.AttackAttr, s.RangedWeapon, s.ItemAttack = e.Variance, e.AttackAttr, e.RangedWeapon, e.ItemAttack
	s.DefenseAttr, s.Range, s.AttackSpeed = e.DefenseAttr, e.Range, e.AttackSpeed
	s.SPCostCut, s.MPCostCut, s.MoveSpeed, s.Disabled = e.SPCostCut, e.MPCostCut, e.MoveSpeed, e.Disabled
	s.CooldownCut = e.CooldownCut
	s.SitRecover = e.SitRecoverPct

	kind := attackMelee
	if e.RangedWeapon {
		kind = attackRanged
	}
	s.MinAttack = uint32(s.Attack[kind])
	s.MaxAttack = satAdd(s.MinAttack, s.Variance)
	s.MinMagic = uint32(s.Attack[attackMagic])
	s.MaxMagic = satAdd(s.MinMagic, s.Variance)
	s.Defense = uint32(s.Guard[attackMelee])
	s.Resistance = uint32(s.Guard[attackMagic])
}

// mobCombat contains the combat values derived from a monster's catalog and
// active effects. Its attack templates supply the weapon part separately.
type mobCombat struct {
	DamageFactor float32
	Level        int32
	Accuracy     [3]int64
	Evasion      [3]int64
	Guard        [3]int32
	Attack       [3]int32
	Critical     int64
	Luc          int64 // its LUC lowers the critical chance of whoever hits it
	Attrib       uint8
}

func mobCombatFor(d *db.MobDef) mobCombat {
	m := mobCombat{Level: int32(d.Level), Attrib: d.Attrib, Critical: int64(d.Luc) * 20, Luc: int64(d.Luc)}
	setCombatChances(&m.Accuracy, &m.Evasion, uint32(d.Dex), uint32(d.Wis), [3]CategoryMods{})
	m.Guard[attackMelee] = int32(d.Defense)
	m.Guard[attackRanged] = boundedCombatStat(int64(d.Defense) + int64(d.ShieldDefense))
	m.Guard[attackMagic] = int32(d.Magic)
	m.Attack[attackMagic] = int32(d.Wis) + int32(d.Level)
	return m
}

func rollCombatChance(chance int64) bool {
	return chance >= fullChance || chance > 0 && rand.Int64N(fullChance) < chance
}

func effectiveEvasion(defenderEvasion, attackerAccuracy int64) int64 {
	return min(max(defenderEvasion-attackerAccuracy, 0), fullChance)
}

// mobLevelPenalty is the percentage adjustment for a monster above a player.
func mobLevelPenalty(playerLevel uint16, mobLevel int32) int64 {
	return int64(min(max(mobLevel-int32(playerLevel), 0), 10)) * 5
}

func playerHitChance(s *CharacterStats, m *mobCombat, kind int) int64 {
	hit := fullChance - effectiveEvasion(m.Evasion[kind], s.Accuracy[kind])
	return hit * (100 - mobLevelPenalty(s.Level, m.Level)) / 100
}

func playerDodgeChance(s *CharacterStats, m *mobCombat, kind int) int64 {
	evasion := effectiveEvasion(s.Evasion[kind], m.Accuracy[kind])
	return evasion * (100 - mobLevelPenalty(s.Level, m.Level)) / 100
}

func playerHits(s *CharacterStats, m *mobCombat, kind int) bool {
	return rollCombatChance(playerHitChance(s, m, kind))
}

func mobHits(m *mobCombat, s *CharacterStats, kind int) bool {
	return rollCombatChance(fullChance - playerDodgeChance(s, m, kind))
}

func userHits(att, def *CharacterStats, kind int) bool {
	return rollCombatChance(fullChance - effectiveEvasion(def.Evasion[kind], att.Accuracy[kind]))
}

// skillSuccess retains a definition's independent success roll. Accuracy
// and evasion still apply, even to skills with a guaranteed catalog roll.
func skillSuccess(def *db.SkillDef) bool {
	return def.SuccessType < 1 || def.SuccessType > 3 || rollCombatChance(int64(def.SuccessValue)*100)
}

func playerSkillHits(s *CharacterStats, m *mobCombat, md *db.MobDef, def *db.SkillDef) bool {
	if state := int(def.StateType); state > 0 && state <= len(md.ResistState) && md.ResistState[state-1] != 0 {
		return false
	}
	for _, id := range md.ResistSkill {
		if id != 0 && id == def.SkillID {
			return false
		}
	}
	return skillSuccess(def) && playerHits(s, m, int(def.Kind))
}

func mobSkillHits(m *mobCombat, def *db.SkillDef, s *CharacterStats) bool {
	return skillSuccess(def) && mobHits(m, s, int(def.Kind))
}

func userSkillHits(att, def *CharacterStats, sk *db.SkillDef) bool {
	return skillSuccess(sk) && userHits(att, def, int(sk.Kind))
}

type hitDamage struct {
	attack       int64
	variance     uint32
	guard        int32
	critical     int64
	targetLuc    int64 // each point of the target's LUC removes 0.01 percentage points
	attackAttr   uint8
	defenseAttr  uint8
	levelPercent int64
}

// criticalChance is the attacker's critical chance (0.2 points per LUC plus
// bonuses) less 0.01 percentage points per point of the target's LUC.
func criticalChance(critical, targetLuc int64) int64 {
	return min(max(critical-targetLuc, 0), fullChance)
}

// weaponHit computes the same damage for all three categories. Using the
// element table's exact tenths keeps truncation at the end of the calculation.
func (h hitDamage) weaponHit(bonus, multiplier int64) (int32, bool) {
	attack := h.attack + bonus
	if h.variance != 0 {
		attack += int64(rand.Uint64N(uint64(h.variance) + 1))
	}
	crit := rollCombatChance(criticalChance(h.critical, h.targetLuc))
	critHalves := int64(2)
	if crit {
		critHalves = 3
	}
	elementTenths := int64(element.Multiplier(h.attackAttr, h.defenseAttr)*10 + .5)
	// Even at the lowest element and level multipliers, this much base
	// damage reaches the ceiling. Bounding it also bounds every product.
	base := min(max(attack-int64(h.guard), 0), maxDamage*5)
	amount := base * elementTenths * critHalves * multiplier * h.levelPercent / 2000
	return int32(min(max(amount, 1), maxDamage)), crit
}

// directAmount scales nonzero immediate skill damage and preserves zero
// components. It is also used for the level adjustment on extra impacts.
func directAmount(amount, levelPercent, limit int64) int32 {
	if amount <= 0 {
		return 0
	}
	return int32(min(max(amount*levelPercent/100, 1), limit))
}

func extraSkillDamage(def *db.SkillDef, levelPercent int64) [3]uint32 {
	values := [3]int32{def.AddDamageHP, def.AddDamageSP, def.AddDamageMP}
	var damage [3]uint32
	for i, value := range values {
		damage[i] = uint32(directAmount(int64(uint16(value)), levelPercent, math.MaxUint16))
	}
	return damage
}

type damagePools [3]uint32 // HP, SP, MP

func currentDamagePools(s *CharacterStats) damagePools {
	return damagePools{s.CurrentHP, s.CurrentSP, s.CurrentMP}
}

func maximumDamagePools(s *CharacterStats) damagePools {
	return damagePools{s.MaxHP, s.MaxSP, s.MaxMP}
}

func (h hitDamage) skillHit(def *db.SkillDef, caster, current, maximum damagePools) ([3]int32, bool) {
	values := [3]int64{int64(uint16(def.DamageHP)), int64(uint16(def.DamageSP)), int64(uint16(def.DamageMP))}
	var damage [3]int32
	switch def.DamageType {
	case 0, 3, 4, 5:
		for i, value := range values {
			if value == 0 {
				continue
			}
			limit := int64(maxDamage)
			switch def.DamageType {
			case 0:
				limit = math.MaxUint16
			case 3:
				value = max(int64(caster[i])*value, 1)
			case 4:
				value = max(int64(maximum[i])*value/100, 1)
			case 5:
				value = max(int64(current[i])*value/100, 1)
			}
			damage[i] = directAmount(value, h.levelPercent, limit)
		}
		return damage, false
	}
	if def.AttrType != 0 {
		h.attackAttr = def.AttrType
	}
	if def.TypeDetail == detailNoElement {
		h.defenseAttr = 0
	}
	var bonus int64
	multiplier := int64(1)
	switch def.DamageType {
	case 1:
		bonus = values[0]
		for i := 1; i < len(damage); i++ {
			damage[i] = directAmount(values[i], h.levelPercent, maxDamage)
		}
	case 2:
		multiplier = max(values[0], 1)
	}
	hp, crit := h.weaponHit(bonus, multiplier)
	damage[0] = hp
	return damage, crit
}

func characterHitDamage(s *CharacterStats, kind int, guard int32, defenseAttr uint8, levelPercent, targetLuc int64) hitDamage {
	return hitDamage{
		attack: int64(s.Attack[kind]), variance: s.Variance, guard: guard, critical: s.Critical, targetLuc: targetLuc,
		attackAttr: s.AttackAttr, defenseAttr: defenseAttr, levelPercent: levelPercent,
	}
}

func playerDamage(s *CharacterStats, m *mobCombat, kind int) (int32, bool) {
	h := characterHitDamage(s, kind, m.Guard[kind], m.Attrib, 100-mobLevelPenalty(s.Level, m.Level), m.Luc)
	return h.weaponHit(0, 1)
}

func playerSkillDamage(s *CharacterStats, m *mobCombat, def *db.SkillDef, current, maximum damagePools) ([3]int32, bool) {
	kind := int(def.Kind)
	h := characterHitDamage(s, kind, m.Guard[kind], m.Attrib, 100-mobLevelPenalty(s.Level, m.Level), m.Luc)
	return h.skillHit(def, currentDamagePools(s), current, maximum)
}

func userDamage(att, def *CharacterStats, kind int) (int32, bool) {
	h := characterHitDamage(att, kind, def.Guard[kind], def.DefenseAttr, 100, int64(def.Luc))
	return h.weaponHit(0, 1)
}

func userSkillDamage(att, def *CharacterStats, sk *db.SkillDef, caster damagePools) ([3]int32, bool) {
	kind := int(sk.Kind)
	h := characterHitDamage(att, kind, def.Guard[kind], def.DefenseAttr, 100, int64(def.Luc))
	return h.skillHit(sk, caster, currentDamagePools(def), maximumDamagePools(def))
}

func mobHitDamage(m *mobCombat, a *db.MobAttack, s *CharacterStats, kind int, attr uint8) hitDamage {
	attack := int64(a.Attack) + int64(m.Attack[kind])
	if m.DamageFactor > 0 {
		attack = int64(float64(attack) * float64(m.DamageFactor))
	}
	return hitDamage{
		attack: attack, variance: uint32(a.Plus), guard: s.Guard[kind], critical: m.Critical, targetLuc: int64(s.Luc),
		attackAttr: attr, defenseAttr: s.DefenseAttr, levelPercent: 100 + mobLevelPenalty(s.Level, m.Level),
	}
}

func mobDamage(m *mobCombat, a *db.MobAttack, s *CharacterStats) (int32, bool) {
	return mobHitDamage(m, a, s, attackMelee, 0).weaponHit(0, 1)
}

func mobSkillDamage(m *mobCombat, md *db.MobDef, def *db.SkillDef, s *CharacterStats, mobHP uint32) ([3]int32, bool) {
	slot := &md.Attacks[0]
	h := mobHitDamage(m, slot, s, int(def.Kind), slot.Attrib)
	return h.skillHit(def, damagePools{mobHP}, currentDamagePools(s), maximumDamagePools(s))
}

// f32 preserves the original periodic-damage calculation's input precision.
func f32(v int32) float64 { return float64(float32(v)) }
