package scene

// SkillActionSound follows the common cast/release branches in 0x5780a0,
// 0x57bdc0 and 0x57e320. IDs 0x21..0x26 are registered at 0x4264b6..0x42653b.
// Physical attacks use separate weapon/voice cues, not a generic spell sound.
func SkillActionSound(action int) string {
	switch action {
	case 85:
		return "ch_maready01.ogg"
	case 87:
		return "ef_in_firing01.ogg"
	case 88:
		return "ch_maready02.ogg"
	case 90:
		return "ch_maattack02.ogg"
	case 91:
		return "ch_buready01.ogg"
	case 93:
		return "ch_buattack01.ogg"
	}
	return ""
}

// SkillReleaseSounds lists the extra cues Game.exe plays when a skill is
// released, besides the cast/release action cue (EP5.4 0x57cd80 and 0x57d7c0,
// the same strings exist in EP4). The animation byte 100..110 of a damaging
// skill selects ch_skill001/003/004/005/008 (101, 105, 106 and 108..110 are
// silent); a debuff with the physical attack kind always uses ch_skill001;
// every physical skill released with a javelin, bow or crossbow adds
// ef_cst_crobow01. The weapon is the caster's item type.
func SkillReleaseSounds(animation uint16, attack, effect, weapon uint8) []string {
	if effect == 3 && attack == 1 {
		return []string{"ch_skill001.ogg"}
	}
	if (effect != 1 && effect != 3 && effect != 5) || attack == 0 {
		return nil
	}
	var out []string
	switch uint8(animation) {
	case 100:
		out = append(out, "ch_skill001.ogg")
	case 102:
		out = append(out, "ch_skill003.ogg")
	case 103:
		out = append(out, "ch_skill004.ogg")
	case 104:
		out = append(out, "ch_skill005.ogg")
	case 107:
		out = append(out, "ch_skill008.ogg")
	}
	if attack == 1 || attack == 2 {
		switch weaponKind(weapon) {
		case 11, 13, 14:
			out = append(out, "ef_cst_crobow01.ogg")
		}
	}
	return out
}

// HealImpactSound is the cue Game.exe 0x57d896 plays on the healed target
// when a heal skill (effect 2) lands as a normal or critical result.
const HealImpactSound = "ef_dmg_heal01.ogg"
