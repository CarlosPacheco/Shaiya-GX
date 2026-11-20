package sdata

import (
	"bytes"
	"encoding/binary"
	"fmt"
)

// RepairNPCSkillCatalog updates three known NPC skill definitions. The
// expected names keep this patch out of other episodes and the player table.
func RepairNPCSkillCatalog(raw []byte) ([]byte, SkillCatalogReport, error) {
	var report SkillCatalogReport
	groups, slots, err := readSkillCatalog(raw)
	if err != nil {
		return nil, report, err
	}
	for _, patch := range []struct {
		id          int
		name        string
		description string
	}{
		{100, "Porcine Party", "Turns enemies within 35 m of the caster into pigs for 15 seconds."},
		{172, "Ice Stun", "Deals water damage equal to 25% of each target's maximum HP and stuns enemies within 35 m of the target for 15 seconds."},
		{344, "Wide Death Sentence", "Deals 70% of maximum HP to enemies within 6 m of the target. The curse kills affected enemies after 20 seconds unless dispelled."},
	} {
		index := (patch.id - 1) * skillSlotsPerID
		if index >= len(slots) || slots[index].rec[0] != 1 || slots[index].name != patch.name {
			return nil, report, fmt.Errorf("NPC skill %d rank 1 should be %s", patch.id, patch.name)
		}
		s := slots[index]
		updated := bytes.Clone(s.rec)
		switch patch.id {
		case 100:
			binary.LittleEndian.PutUint16(updated[112:], 101) // ChangeType: use the monster in ChangeLevel.
		case 172:
			updated[21], updated[67], updated[74] = 3, 4, 0 // Debuff, damage from max HP, no periodic damage.
			binary.LittleEndian.PutUint16(updated[22:], 64) // TypeDetail: stun.
			binary.LittleEndian.PutUint16(updated[68:], 25) // DamageHP is a percentage.
			binary.LittleEndian.PutUint16(updated[75:], 0)  // TimeDamageHP.
		case 344:
			updated[21] = 3                                 // TypeEffect: apply the death sentence as a debuff.
			binary.LittleEndian.PutUint16(updated[59:], 20) // KeepTime in seconds.
		}
		if s.desc != patch.description || !bytes.Equal(s.rec, updated) {
			s.rec, s.desc, s.descRaw = updated, patch.description, lengthPrefixed(patch.description)
			report.NPCSkills = append(report.NPCSkills, s.key())
		}
	}
	if len(report.NPCSkills) == 0 {
		return raw, report, nil
	}
	plain, err := writeSkillCatalog(groups, slots)
	if err != nil {
		return nil, report, err
	}
	out, err := EncodeLike(raw, plain)
	return out, report, err
}
