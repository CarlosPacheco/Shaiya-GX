package app

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/Spelunkern/Shaiya-GX/internal/data/sdata"
	"github.com/Spelunkern/Shaiya-GX/internal/skillchance"
)

var skillTextTypos = strings.NewReplacer(
	"near by", "nearby", "Near by", "Nearby", "wihtin", "within",
	"addtional", "additional", "untill", "until", "efffect", "effect",
	"taregt", "target", "memebers", "members", "secodns", "seconds",
	"m eters", "meters", "for 1 seconds", "for 1 second", "a 80 percent", "an 80 percent", "a 100 percent certain", "a 100 percent chance",
)

var skillTextRadius = regexp.MustCompile(`(?i)\bwithin\s+\d+\s+(?:meters?|m)\b`)

func refreshSkillTooltips(defs map[sdata.SkillKey]sdata.Skill, npc bool) {
	for key, d := range defs {
		if key.Level == 0 || d.Name == "" {
			continue
		}
		d.Name = strings.Join(strings.Fields(d.Name), " ")
		d.Description = skillTextTypos.Replace(strings.TrimSpace(d.Description))
		if npc {
			refreshNPCTooltip(key.ID, &d)
		} else {
			refreshPlayerTooltip(key, &d)
		}
		defs[key] = d
	}
	if npc {
		return
	}
	for key, d := range defs {
		if key.Level == 0 || d.Effect == 0 {
			continue
		}
		previous := defs[sdata.SkillKey{ID: d.PreviousSkill, Level: 1}]
		d.Description = sdata.SkillCooldownText(d.Description, previous.Name)
		defs[key] = d
	}
}

func refreshPlayerTooltip(key sdata.SkillKey, d *sdata.Skill) {
	duration := d.KeepSeconds()
	radius := d.ApplyRange
	switch {
	case d.Effect == 2 && (d.Detail == 41 && d.Heal[0] > 0 || d.Detail == 42):
		subject := "the target"
		switch d.Target {
		case 2:
			subject = "you"
		case 4, 5:
			subject = fmt.Sprintf("you and party members within %d m", radius)
		case 8:
			subject = fmt.Sprintf("you and raid members within %d m", radius)
		}
		if d.Detail == 42 {
			d.Description = "Removes harmful effects from " + subject + "."
			return
		}
		recovery := []string{fmt.Sprintf("%d HP plus twice your WIS", d.Heal[0])}
		for i := 1; i < len(d.Heal); i++ {
			if d.Heal[i] > 0 {
				recovery = append(recovery, fmt.Sprintf("%d %s", d.Heal[i], skillResources[i]))
			}
		}
		d.Description = fmt.Sprintf("Restores %s to %s.", skillTextList(recovery), subject)
	case d.Effect == 4 && d.Detail == 81 && duration > 0:
		var recovery []string
		for i, amount := range d.TimeHeal {
			if amount > 0 {
				recovery = append(recovery, fmt.Sprintf("%d %s", amount, skillResources[i]))
			}
		}
		if len(recovery) > 0 {
			d.Description = fmt.Sprintf("Restores %s every 3 seconds for %s.", skillTextList(recovery), skillDuration(duration))
			if key.ID == 278 || key.ID == 279 {
				resource := "Stamina"
				if key.ID == 279 {
					resource = "Mana"
				}
				d.Name = fmt.Sprintf("Continuous %s Recovery Lv%d", resource, key.Level)
			}
		}
	case key.ID == 33 && d.Effect == 3 && d.Detail == 62 && d.AbilityType == [3]uint8{17, 16}:
		d.Description = fmt.Sprintf("Lowers the target's movement speed by %s and attack speed by %s for %s.",
			skillTiers(effectiveSkillAbility(17, d.AbilityValue[0])), skillTiers(effectiveSkillAbility(16, d.AbilityValue[1])), skillDuration(duration))
		if d.SuccessType >= 1 && d.SuccessType <= 3 && d.SuccessValue < 100 {
			d.Description += fmt.Sprintf(" Base chance: %d%%.", d.SuccessValue)
		}
	case key.ID == 40 && d.Effect == 3 && d.Detail == 62 && d.Target == 6 && d.AbilityType == [3]uint8{16}:
		d.Description = fmt.Sprintf("Smashes the ground with a one-handed weapon, two-handed weapon, spear, or blunt weapon. Deals melee damage to enemies within %d m of you and lowers their attack speed by %s for %s.",
			radius, skillTiers(effectiveSkillAbility(16, d.AbilityValue[0])), skillDuration(duration))
	case key.ID == 43 && d.Detail == 93:
		d.Description = fmt.Sprintf("Reveals stealthed enemy players within %d m of you for %s.", radius, skillDuration(duration))
	case key.ID == 47 && d.Effect == 4 && d.Detail == 84:
		// The server's Fire Thorn has a fixed 6 m reach, independent of ApplyRange.
		d.Description = fmt.Sprintf("Deals %d damage every 2 seconds to enemies within 6 m of you and draws monster aggro. Lasts %s.", d.Damage[0], skillDuration(duration))
	case key.ID == 48 && d.Effect == 5 && d.Detail == 102:
		d.Description = fmt.Sprintf("Creates a field for %s that deals %d damage every half second to enemies within %d m.", skillDuration(duration), d.Damage[0], radius)
	case key.ID == 74 && d.Effect == 4 && d.Detail == 87 && key.Level <= 3:
		pools := [...]string{"SP", "MP", "SP, then MP"}[key.Level-1]
		d.Description = fmt.Sprintf("For %s, spends %s to absorb incoming hit damage point for point before it reduces HP.", skillDuration(duration), pools)
	case key.ID == 105 && d.Effect == 4 && d.Detail == 69:
		d.Description = fmt.Sprintf("Forces hostile monsters within %d m to target you and increases your threat based on REC.", radius)
	case key.ID == 140 && d.Effect == 4 && d.Detail == 82 && d.AbilityType == [3]uint8{15}:
		d.Description = fmt.Sprintf("Increases the range of your basic attacks by %d m for %s.", effectiveSkillAbility(15, d.AbilityValue[0]), skillDuration(duration))
	case key.ID == 154 && d.Effect == 1 && d.DamageType == 0:
		d.Description = fmt.Sprintf("Deals %d HP damage with a jumping bow shot that ignores defense.", d.Damage[0])
	case key.ID == 160 && d.Effect == 3 && d.Detail == 62 && d.AbilityType == [3]uint8{29}:
		d.Name = fmt.Sprintf("Arcane Range Lv%d", key.Level)
		d.Description = fmt.Sprintf("Lowers melee evasion by %d percentage points for %s on enemies within %d m of the target.",
			skillchance.Modifier(29, effectiveSkillAbility(29, d.AbilityValue[0])), skillDuration(duration), radius)
	case key.ID == 179 && d.Effect == 4 && d.Detail == 88 && d.Target == 5 && radius > 0:
		d.Description = fmt.Sprintf("Grants stealth to you and party members within %d m for up to %s.", radius, skillDuration(duration))
	case key.ID == 309 && d.Effect == 1 && d.DamageType == 2:
		d.Description, _ = skillTooltipDamage(d)
		d.Description += fmt.Sprintf(" Base chance: %d%%.", d.SuccessValue)
	case key.ID == 310 && d.Effect == 3 && d.Detail == 66:
		d.Description, _ = skillDebuffText(d)
	case d.Effect == 3 && (d.Detail == 61 || d.Detail == 63) && duration > 0:
		attack, ok := skillTooltipDamage(d)
		if !ok {
			return
		}
		status, ok := skillDebuffText(d)
		if !ok || status == "" {
			return
		}
		d.Description = strings.TrimSpace(attack + " " + status + " " + skillAreaText(d))
		if key.ID == 138 {
			d.Description += " Generates monster threat."
		}
	case radius > 0 && playerTooltipHasArea(d):
		// The catalog has old diameter labels and stale rank-specific ranges.
		// Only rewrite descriptions that contain a single explicit area.
		if len(skillTextRadius.FindAllStringIndex(d.Description, 2)) == 1 {
			d.Description = skillTextRadius.ReplaceAllString(d.Description, fmt.Sprintf("within %d m", radius))
		}
	}
}

// Like GX Server's skill definition loader, a present ability with a
// zero catalog value contributes one point.
func effectiveSkillAbility(kind uint8, value int16) int32 {
	if kind != 0 && value == 0 {
		return 1
	}
	return int32(value)
}

func playerTooltipHasArea(d *sdata.Skill) bool {
	switch d.Effect {
	case 1, 3:
		return d.Target == 6 || d.Target == 7
	case 2, 4:
		return d.Target == 4 || d.Target == 5 || d.Target == 8
	default:
		return false
	}
}

var skillResources = [...]string{"HP", "SP", "MP"}

func skillDuration(seconds uint32) string {
	if seconds == 1 {
		return "1 second"
	}
	return fmt.Sprintf("%d seconds", seconds)
}

func skillTiers(tiers int32) string {
	if tiers == 1 {
		return "1 tier"
	}
	return fmt.Sprintf("%d tiers", tiers)
}

func skillTextList(parts []string) string {
	if len(parts) <= 1 {
		return strings.Join(parts, "")
	}
	return strings.Join(parts[:len(parts)-1], ", ") + " and " + parts[len(parts)-1]
}
