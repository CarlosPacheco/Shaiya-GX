package sdata

import (
	"encoding/binary"
	"fmt"
)

// UpdateCombatDebuffTooltips corrects only the Slump Shot and Confusion descriptions.
func UpdateCombatDebuffTooltips(raw []byte) ([]byte, SkillCatalogReport, error) {
	var report SkillCatalogReport
	groups, slots, err := readSkillCatalog(raw)
	if err != nil {
		return nil, report, err
	}
	updateCombatDebuffText(slots, &report)
	if report.TextFixes == 0 {
		return raw, report, nil
	}
	out, err := writeSkillCatalog(groups, slots)
	return out, report, err
}

func updateCombatDebuffText(slots []*skillSlot, report *SkillCatalogReport) {
	for _, s := range slots {
		if (s.id != 51 && s.id != 96) || s.rec[0] == 0 || s.dead() {
			continue
		}
		text := combatDebuffDescription(s)
		if text != "" && text != s.desc {
			s.desc, s.descRaw = text, lengthPrefixed(text)
			report.TextFixes++
		}
	}
}

func combatDebuffDescription(s *skillSlot) string {
	p := s.rec
	const abilityDown = 62
	if p[21] != 3 || binary.LittleEndian.Uint16(p[22:]) != abilityDown || p[67] != 1 {
		return ""
	}
	seconds := (Skill{KeepTime: binary.LittleEndian.Uint16(p[59:]), FixRange: p[111]}).KeepSeconds()
	ability := [3]uint8{p[87], p[90], p[93]}
	first := int16(binary.LittleEndian.Uint16(p[88:]))
	second := int16(binary.LittleEndian.Uint16(p[91:]))
	third := int16(binary.LittleEndian.Uint16(p[94:]))
	if s.id == 51 && p[20] == 2 && p[56] == 3 {
		switch ability {
		case [3]uint8{4, 5, 0}:
			return fmt.Sprintf("Deals ranged damage. For %d seconds, lowers STR by %d and REC by %d on enemy players.", seconds, first, second)
		case [3]uint8{4, 5, 16}:
			return fmt.Sprintf("Deals ranged damage. For %d seconds, lowers attack speed by %d tier on players and monsters, and lowers STR by %d and REC by %d on enemy players only.", seconds, third, first, second)
		}
	}
	if s.id == 96 && p[20] == 3 && p[56] == 6 && ability == ([3]uint8{5, 8, 7}) {
		bonus := binary.LittleEndian.Uint16(p[68:])
		return fmt.Sprintf("Deals magic damage with a bonus of %d to enemies within %d m. For %d seconds, lowers DEX by %d and WIS by %d on players and monsters, and lowers REC by %d on enemy players only.", bonus, p[57], seconds, second, third, first)
	}
	return ""
}
