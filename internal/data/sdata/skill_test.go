package sdata

import (
	"encoding/binary"
	"testing"
)

func FuzzSkills(f *testing.F) {
	f.Add([]byte{0, 0, 0, 0})
	f.Fuzz(func(t *testing.T, b []byte) { ParseSkills(b) })
}

func TestSkillIDsAreOneBased(t *testing.T) {
	// One group, nine empty-string records with their declared levels.
	b := make([]byte, 4+9*124)
	b[0] = 1
	for i := 0; i < 9; i++ {
		b[4+i*124+8] = byte(i + 1)
	}
	b[12+47] = 24
	b[12+49] = 4
	b[12+3], b[12+4], b[12+5], b[12+6] = 100, 98, 2, 37
	b[12+22] = 121
	b[12+59], b[12+60], b[12+111] = 2, 0, 1
	b[12+87], b[12+88], b[12+89], b[12+110] = 18, 244, 255, 20
	p := b[12:]
	binary.LittleEndian.PutUint16(p[52:], 135)
	p[58], p[67], p[74] = 3, 4, 5
	for i, off := range []int{68, 75, 81, 96, 102} {
		for j := range 3 {
			binary.LittleEndian.PutUint16(p[off+2*j:], uint16(i*3+j+1))
		}
	}
	defs, err := ParseSkills(b)
	if err != nil {
		t.Fatal(err)
	}
	if len(defs) != 9 || defs[SkillKey{1, 1}].Level != 1 {
		t.Fatal("group did not map to wire skill 1")
	}
	if _, exists := defs[SkillKey{0, 1}]; exists {
		t.Fatal("zero-based wire skill ID")
	}
	if d := defs[SkillKey{1, 1}]; d.AttackRange != 24 || d.Attribute != 4 || d.Animation != 100 || d.VisualCode != 610 || d.Detail != 121 || d.Toggle != 2 || d.Sound != 37 {
		t.Fatalf("EP5 visual fields: %+v", d)
	}
	if d := defs[SkillKey{1, 1}]; d.KeepTime != 2 || d.FixRange != 1 || d.KeepSeconds() != 7200 {
		t.Fatal("EP5 duration fields", d)
	}
	if d := defs[SkillKey{1, 1}]; d.AbilityType[0] != 18 || d.AbilityValue[0] != -12 || d.LimitHP != 20 {
		t.Fatal("EP5 ability fields", d)
	}
	d := defs[SkillKey{1, 1}]
	if d.PreviousSkill != 135 || d.MultiAttack != 3 || d.DamageType != 4 || d.TimeDamageType != 5 ||
		d.Damage != [3]uint16{1, 2, 3} || d.TimeDamage != [3]uint16{4, 5, 6} ||
		d.ExtraDamage != [3]uint16{7, 8, 9} || d.Heal != [3]int16{10, 11, 12} ||
		d.TimeHeal != [3]int16{13, 14, 15} {
		t.Fatalf("EP5 tooltip fields: %+v", d)
	}
}
