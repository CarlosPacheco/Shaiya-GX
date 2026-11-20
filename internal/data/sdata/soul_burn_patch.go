package sdata

import (
	"bytes"
	"encoding/binary"
	"fmt"
)

const soulBurnSkillID = 125

const soulBurnDescription = "Forces a dead enemy player to respawn immediately at their bind point."

// EnableSoulBurn enables only skill 125 in an existing player catalog.
func EnableSoulBurn(raw []byte) ([]byte, SkillCatalogReport, error) {
	var report SkillCatalogReport
	groups, slots, err := readSkillCatalog(raw)
	if err != nil {
		return nil, report, err
	}
	if !enableSoulBurnSlots(slots, &report) {
		return nil, report, fmt.Errorf("Soul Burn level 1 is missing from the player skill catalog")
	}
	if len(report.SoulBurnLevels) == 0 {
		return raw, report, nil
	}
	out, err := writeSkillCatalog(groups, slots)
	return out, report, err
}

func enableSoulBurnSlots(slots []*skillSlot, report *SkillCatalogReport) bool {
	index := (soulBurnSkillID - 1) * skillSlotsPerID
	if len(slots) < index+skillSlotsPerID {
		return false
	}
	group := slots[index : index+skillSlotsPerID]
	first := group[0]
	if first.rec[0] != 1 || binary.LittleEndian.Uint16(first.rec[22:]) != 122 {
		return false
	}
	updated := bytes.Clone(first.rec)
	updated[16] = 1                                 // Defensemage covers Priests and Oracles.
	updated[18], updated[19], updated[21] = 3, 4, 6 // SkillPoint, TypeShow, TypeEffect
	updated[54], updated[55] = 1, 100               // SuccessType, SuccessValue
	binary.LittleEndian.PutUint16(updated[59:], 0)  // KeepTime: the effect is immediate.
	if !bytes.Equal(first.rec, updated) || first.desc != soulBurnDescription {
		report.SoulBurnLevels = append(report.SoulBurnLevels, first.key())
		first.rec, first.desc, first.changed = updated, soulBurnDescription, true
	}
	for _, s := range group[1:3] {
		if s.rec[0] != 0 && binary.LittleEndian.Uint16(s.rec[22:]) == 122 {
			report.SoulBurnLevels = append(report.SoulBurnLevels, s.key())
			s.name, s.desc, s.rec, s.changed = "", "", make([]byte, skillRecordSize), true
		}
	}
	return true
}
