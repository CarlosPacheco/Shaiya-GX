package app

import (
	"fmt"
	"strings"

	"github.com/Spelunkern/Shaiya-GX/internal/data/sdata"
	"github.com/Spelunkern/Shaiya-GX/internal/skillchance"
)

func refreshNPCTooltip(id uint16, d *sdata.Skill) {
	switch id {
	case 3:
		if d.Name == "Silent" {
			d.Name = "Silence"
		}
	case 253:
		if d.Name == "Excrutiating Pain" {
			d.Name = "Excruciating Pain"
		}
	}
	// Both normal mob attack slots and AI-triggered heal skills heal their caster.
	if d.Effect == 2 && d.Heal[0] > 0 {
		d.Description = fmt.Sprintf("Restores %d HP to the caster.", d.Heal[0])
		return
	}
	damage, understood := skillTooltipDamage(d)
	if !understood {
		return
	}
	var status string
	switch d.Effect {
	case 1:
		switch d.Detail {
		case 21, 25:
		case 22:
			status = fmt.Sprintf("Hits up to %d times.", min(d.MultiAttack, 60))
		case 23:
			status = "The caster dies after the attack."
		default:
			return
		}
	case 3:
		status, understood = skillDebuffText(d)
		if !understood {
			return
		}
	default:
		return
	}
	var parts []string
	if damage != "" {
		parts = append(parts, damage)
	}
	if status != "" {
		parts = append(parts, status)
	}
	if len(parts) == 0 {
		return
	}
	if area := skillAreaText(d); area != "" {
		parts = append(parts, area)
	}
	d.Description = strings.Join(parts, " ")
}

func skillAreaText(d *sdata.Skill) string {
	switch d.Target {
	case 6:
		return fmt.Sprintf("Area: %d m around the caster.", d.ApplyRange)
	case 7:
		return fmt.Sprintf("Area: %d m around the target.", d.ApplyRange)
	default:
		return ""
	}
}

func skillTooltipDamage(d *sdata.Skill) (string, bool) {
	var damage []string
	for i, amount := range d.Damage {
		if amount == 0 {
			continue
		}
		switch d.DamageType {
		case 0, 1:
			if d.DamageType == 1 && i == 0 {
				continue // The HP value is an attack bonus in this mode.
			}
			damage = append(damage, fmt.Sprintf("%d %s", amount, skillResources[i]))
		case 4, 5:
			damage = append(damage, skillPercentAmount(amount, d.DamageType, skillResources[i]))
		case 2:
			continue
		case 3:
			damage = append(damage, fmt.Sprintf("%d times the caster's current %s", amount, skillResources[i]))
		default:
			return "", false
		}
	}
	var parts []string
	switch d.DamageType {
	case 0, 3, 4, 5:
	case 1, 2:
		kind := [...]string{"melee", "ranged", "magic"}[int(max(1, min(d.Attack, 3)))-1]
		attack := "Deals " + kind + " damage"
		switch {
		case d.DamageType == 1 && d.Damage[0] > 0:
			attack += fmt.Sprintf(" with a bonus of %d", d.Damage[0])
		case d.DamageType == 2 && d.Damage[0] > 1:
			attack += fmt.Sprintf(" with a ×%d attack multiplier", d.Damage[0])
		}
		parts = append(parts, attack+".")
	default:
		return "", false
	}
	if len(damage) > 0 {
		parts = append(parts, "Deals damage equal to "+skillTextList(damage)+".")
	}
	var extra []string
	for i, amount := range d.ExtraDamage {
		if amount > 0 {
			extra = append(extra, fmt.Sprintf("%d %s", amount, skillResources[i]))
		}
	}
	if len(extra) > 0 {
		parts = append(parts, "Deals an additional "+skillTextList(extra)+" damage.")
	}
	return strings.Join(parts, " "), true
}

func skillPercentAmount(amount uint16, mode uint8, resource string) string {
	pool := "maximum"
	if mode == 5 {
		pool = "current"
	}
	return fmt.Sprintf("%d%% of the target's %s %s", amount, pool, resource)
}

func skillDebuffText(d *sdata.Skill) (string, bool) {
	duration := skillDuration(d.KeepSeconds())
	switch d.Detail {
	case 21, 25:
		return "", true
	case 61:
		var damage []string
		for i, amount := range d.TimeDamage {
			if amount == 0 {
				continue
			}
			switch d.TimeDamageType {
			case 0:
				damage = append(damage, fmt.Sprintf("%d %s", amount, skillResources[i]))
			case 4, 5:
				damage = append(damage, skillPercentAmount(amount, d.TimeDamageType, skillResources[i]))
			case 3:
				damage = append(damage, fmt.Sprintf("%d times the target's current %s", amount, skillResources[i]))
			case 1, 2:
				// The periodic damage formula does not use these modes.
				continue
			default:
				return "", false
			}
		}
		if len(damage) > 0 {
			return fmt.Sprintf("Deals periodic damage equal to %s every 3 seconds for %s.", skillTextList(damage), duration), true
		}
	case 62, 67:
		var abilities []string
		for i, kind := range d.AbilityType {
			value := effectiveSkillAbility(kind, d.AbilityValue[i])
			if value == 0 {
				continue
			}
			text := npcTooltipAbility(kind, value)
			if text == "" {
				return "", false
			}
			abilities = append(abilities, text)
		}
		if len(abilities) > 0 {
			return fmt.Sprintf("For %s: %s.", duration, skillTextList(abilities)), true
		}
	case 63:
		return fmt.Sprintf("The curse deals damage equal to remaining HP after %s unless removed.", duration), true
	case 64, 65, 66:
		if d.KeepSeconds() == 0 {
			if d.Detail == 66 {
				return "Briefly puts targets to sleep. Damage wakes them.", true
			}
			return "", false
		}
		switch d.Detail {
		case 64:
			return fmt.Sprintf("Stuns targets for %s.", duration), true
		case 65:
			return fmt.Sprintf("Immobilizes targets for %s.", duration), true
		default:
			return fmt.Sprintf("Puts targets to sleep for up to %s. Damage wakes them.", duration), true
		}
	case 68:
		return fmt.Sprintf("Removes the target's defense element for %s.", duration), true
	default:
		return "", false
	}
	return "", true
}

func npcTooltipAbility(kind uint8, value int32) string {
	names := [...]string{
		4: "STR", 5: "REC", 6: "INT", 7: "WIS", 8: "DEX", 9: "LUC",
		10: "HP recovery", 11: "SP recovery", 12: "MP recovery",
		13: "SP cost reduction", 14: "MP cost reduction", 15: "attack range",
		16: "attack speed", 17: "movement speed", 18: "critical chance", 19: "cooldown reduction",
		20: "melee accuracy", 21: "ranged accuracy", 22: "magic accuracy",
		23: "melee attack", 24: "ranged attack", 25: "magic attack",
		26: "melee defense", 27: "ranged defense", 28: "magic resistance",
		29: "melee evasion", 30: "ranged evasion", 31: "magic evasion",
	}
	if kind >= 32 && kind <= 34 && value > 0 {
		return "blocks " + [...]string{"melee attacks", "ranged attacks", "magic skills"}[kind-32]
	}
	if int(kind) >= len(names) || names[kind] == "" {
		return ""
	}
	verb := "lowers"
	if value < 0 {
		verb, value = "raises", -value
	}
	value = skillchance.Modifier(kind, value)
	unit := ""
	switch kind {
	case 16, 17:
		unit = " tiers"
		if value == 1 {
			unit = " tier"
		}
	case 18, 20, 21, 22, 29, 30, 31:
		unit = " percentage points"
	}
	return fmt.Sprintf("%s %s by %d%s", verb, names[kind], value, unit)
}
