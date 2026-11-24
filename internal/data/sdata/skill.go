package sdata

import (
	"encoding/binary"
	"fmt"
	"strings"
)

// Skill: EP5 SkillDefinition in Parsec; nine levels per group, wire ID = group index + 1 (Rapture SkillID).
// The fixed payload is 116 bytes after name and description.
type Skill struct {
	Name, Description                    string
	Level                                uint8
	Icon, RequiredLevel                  uint16
	Animation, Sound                     uint16
	VisualCode, Detail, PreviousSkill    uint16
	Toggle                               uint8
	Country                              uint8
	Classes                              [6]uint8
	Points, Show, Attack, Effect, Target uint8
	SP, MP, ResetTime                    uint16
	ReadyTime                            uint8
	AttackRange, ApplyRange, MultiAttack uint8
	Attribute                            uint8
	SuccessType, SuccessValue            uint8
	KeepTime                             uint16
	FixRange                             uint8
	AbilityType                          [3]uint8
	AbilityValue                         [3]int16
	DamageType, TimeDamageType           uint8
	Damage, TimeDamage, ExtraDamage      [3]uint16 // HP, SP, MP
	Heal, TimeHeal                       [3]int16  // HP, SP, MP
	LimitHP                              uint8
	NeedWeapon                           [15]bool // weapon real types 1..15 that may use it
	NeedShield                           bool
	ChangeType, ChangeLevel              uint16
}
type SkillKey struct {
	ID    uint16
	Level uint8
}

// IsSoulBurn also recognizes the original catalog's pre-normalized effect.
func (s Skill) IsSoulBurn() bool { return s.Detail == 122 && (s.Effect == 3 || s.Effect == 6) }

// RetiredPlayerSkill applies only to the player catalog. NPC skill IDs are independent.
func RetiredPlayerSkill(id uint16) bool {
	switch id {
	case 3, 4, 226, 227, 229, 230, 234, 236, 240, 241, 253, 254, 255, 261, 266, 267, 286:
		return true
	default:
		return false
	}
}

var retiredGrowthText = strings.NewReplacer(
	" (Use this skill in caution with Ultimate Mode.)", "",
	" Only Ultimate Mode can learn this level of skill.", "",
)

func ParseSkills(raw []byte) (map[SkillKey]Skill, error) {
	b, err := Decrypt(raw)
	if err != nil {
		return nil, err
	}
	c := &cursor{b: b}
	n := c.count(9 * 124)
	if n > 65535 {
		return nil, fmt.Errorf("skill: too many groups %d", n)
	}
	out := map[SkillKey]Skill{}
	for id := 0; id < n && c.err == nil; id++ {
		for level := 0; level < 9 && c.err == nil; level++ {
			s := Skill{Name: c.str(), Description: retiredGrowthText.Replace(c.str())}
			p := c.take(116)
			s.Level = p[0]
			s.Icon = binary.LittleEndian.Uint16(p[1:])
			s.Animation = uint16(p[3])
			s.VisualCode = binary.LittleEndian.Uint16(p[4:])
			s.Detail = binary.LittleEndian.Uint16(p[22:])
			s.Toggle, s.Sound = p[5], binary.LittleEndian.Uint16(p[6:])
			s.RequiredLevel = binary.LittleEndian.Uint16(p[8:])
			s.Country = p[10]
			copy(s.Classes[:], p[11:17])
			s.PreviousSkill = binary.LittleEndian.Uint16(p[52:])
			s.SuccessType, s.SuccessValue = p[54], p[55]
			s.Points, s.Show, s.Attack, s.Effect = p[18], p[19], p[20], p[21]
			s.SP, s.MP = binary.LittleEndian.Uint16(p[40:]), binary.LittleEndian.Uint16(p[42:])
			s.ReadyTime, s.ResetTime, s.Target = p[44], binary.LittleEndian.Uint16(p[45:]), p[56]
			s.AttackRange, s.ApplyRange, s.MultiAttack = p[47], p[57], p[58]
			s.Attribute = p[49]
			s.KeepTime, s.FixRange = binary.LittleEndian.Uint16(p[59:]), p[111]
			for i := range s.AbilityType {
				s.AbilityType[i] = p[87+i*3]
				s.AbilityValue[i] = int16(binary.LittleEndian.Uint16(p[88+i*3:]))
			}
			s.DamageType, s.TimeDamageType = p[67], p[74]
			for i := range s.Damage {
				offset := 2 * i
				s.Damage[i] = binary.LittleEndian.Uint16(p[68+offset:])
				s.TimeDamage[i] = binary.LittleEndian.Uint16(p[75+offset:])
				s.ExtraDamage[i] = binary.LittleEndian.Uint16(p[81+offset:])
				s.Heal[i] = int16(binary.LittleEndian.Uint16(p[96+offset:]))
				s.TimeHeal[i] = int16(binary.LittleEndian.Uint16(p[102+offset:]))
			}
			s.LimitHP = p[110]
			for i := range s.NeedWeapon {
				s.NeedWeapon[i] = p[24+i] != 0
			}
			s.NeedShield = p[39] != 0
			s.ChangeType, s.ChangeLevel = binary.LittleEndian.Uint16(p[112:]), binary.LittleEndian.Uint16(p[114:])
			out[SkillKey{uint16(id + 1), s.Level}] = s
		}
	}
	if c.err != nil {
		return nil, c.err
	}
	if c.off != len(b) {
		return nil, fmt.Errorf("skill: %d trailing bytes", len(b)-c.off)
	}
	return out, nil
}

// KeepSeconds follows Rapture's normalized KeepTime/FixRange. A newly applied
// buff lacks a wire duration; snapshots supply their actual remaining seconds.
func (s Skill) KeepSeconds() uint32 {
	n := uint32(s.KeepTime)
	if s.FixRange == 1 {
		n *= 3600
	}
	return n
}
