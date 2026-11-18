package proto

import "fmt"

// GXSkillCooldown tells the client that a skill's reuse timer has started
// without an action of the player: skill uint16, remaining seconds uint16.
// Potential skills use it when they trigger by themselves.
const GXSkillCooldown Opcode = 0x7F16

type SkillCooldown struct{ Skill, Seconds uint16 }

func DecodeSkillCooldown(p []byte) (v SkillCooldown, err error) {
	if len(p) != 6 {
		return v, fmt.Errorf("invalid GX skill cooldown length")
	}
	r := NewReader(p)
	v.Skill, v.Seconds = r.U16(), r.U16()
	return v, r.Err()
}
