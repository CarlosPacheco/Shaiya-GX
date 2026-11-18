package proto

import "fmt"

// GXSkillCooldown tells the client that a skill's reuse timer has started
// by itself: skill uint16, remaining seconds uint16, total seconds uint16.
// Total is what a full timer bar represents; 0 keeps the current one. A
// Potential triggering, a cooldown shortened by a cooldown reduction and a
// reduction applied to a timer that is already running all use it.
const GXSkillCooldown Opcode = 0x7F16

type SkillCooldown struct{ Skill, Seconds, Total uint16 }

func DecodeSkillCooldown(p []byte) (v SkillCooldown, err error) {
	if len(p) != 8 {
		return v, fmt.Errorf("invalid GX skill cooldown length")
	}
	r := NewReader(p)
	v.Skill, v.Seconds, v.Total = r.U16(), r.U16(), r.U16()
	return v, r.Err()
}
