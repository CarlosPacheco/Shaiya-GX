package blessingrules

type Effect struct {
	Column, Label, Unit, Group string
}

const BonusCount = len(Effects)

const deathEXPLossReduction = 25

// Effects follow the original KillStatus type IDs, starting at one.
var Effects = [...]Effect{
	{"MaxHPPct", "Maximum HP", "%", "Attributes"},
	{"MaxMPPct", "Maximum MP", "%", "Attributes"},
	{"MaxSPPct", "Maximum SP", "%", "Attributes"},
	{"StrengthBonus", "Strength", "points", "Attributes"},
	{"ReactionBonus", "Reaction", "points", "Attributes"},
	{"IntelligenceBonus", "Intelligence", "points", "Attributes"},
	{"WisdomBonus", "Wisdom", "points", "Attributes"},
	{"DexterityBonus", "Dexterity", "points", "Attributes"},
	{"LuckBonus", "Luck", "points", "Attributes"},
	{"RestingHPRecoveryPctOfMax", "HP recovery out of combat", "% of maximum HP", "Recovery"},
	{"RestingSPRecoveryPctOfMax", "SP recovery out of combat", "% of maximum SP", "Recovery"},
	{"RestingMPRecoveryPctOfMax", "MP recovery out of combat", "% of maximum MP", "Recovery"},
	{"CombatHPRecoveryPctOfMax", "HP recovery in combat", "% of maximum HP", "Recovery"},
	{"CombatSPRecoveryPctOfMax", "SP recovery in combat", "% of maximum SP", "Recovery"},
	{"CombatMPRecoveryPctOfMax", "MP recovery in combat", "% of maximum MP", "Recovery"},
	{"CriticalChanceBonus", "Critical chance", "points", "Combat"},
	{"MeleeAccuracyBonus", "Melee accuracy", "points", "Combat"},
	{"MeleeDefensePct", "Melee defense", "%", "Combat"},
	{"RangedDefensePct", "Ranged defense", "%", "Combat"},
	{"MagicEvasionLegacyBonus", "Magic evasion (legacy slot)", "points", "Combat"},
	{"MeleeEvasionBonus", "Melee evasion", "points", "Combat"},
	{"RangedEvasionBonus", "Ranged evasion", "points", "Combat"},
	{"MagicEvasionBonus", "Magic evasion", "points", "Combat"},
	{"LapisChanceBonus", "Lapis link / extract-all chance", "percentage points", "Lapis"},
	{"DeathEXPLossReductionPct", "Death EXP loss reduction", "%", "Death penalty"},
	{"UnusedRepairDiscount", "Repair discount (unused)", "raw", "Historical"},
	{"UnusedConsumableCastReduction", "Consumable cast reduction (unused)", "raw", "Historical"},
}
