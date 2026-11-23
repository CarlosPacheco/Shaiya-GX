package sdata

import "fmt"

const enhanceWindSpinSkillID = 156

// AllowSpearEnhanceWindSpin lists the spear (weapon type 6) in Enhance Wind
// Spin's weapon flags. Its description names spears, but the catalog only
// flagged two-handed swords, two-handed axes and dual swords.
func AllowSpearEnhanceWindSpin(raw []byte) ([]byte, SkillCatalogReport, error) {
	var report SkillCatalogReport
	groups, slots, err := readSkillCatalog(raw)
	if err != nil {
		return nil, report, err
	}
	index := (enhanceWindSpinSkillID - 1) * skillSlotsPerID
	if len(slots) < index+skillSlotsPerID {
		return nil, report, fmt.Errorf("Enhance Wind Spin is missing from the skill catalog")
	}
	for _, s := range slots[index : index+skillSlotsPerID] {
		// Weapon flags start at record offset 24; type 6 is the sixth.
		if s.rec[0] != 0 && s.rec[24+5] == 0 {
			s.rec[24+5], s.changed = 1, true
			report.WeaponFlags = append(report.WeaponFlags, s.key())
		}
	}
	if len(report.WeaponFlags) == 0 {
		return raw, report, nil
	}
	out, err := writeSkillCatalog(groups, slots)
	return out, report, err
}
