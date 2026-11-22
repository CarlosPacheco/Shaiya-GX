package proto

import "fmt"

// GXSkillFailure supplies a native SysMsg ID without inventing native result codes.
const GXSkillFailure Opcode = 0x7F0E

const (
	MsgCannotUseSkill = 216
	MsgCannotAttack   = 316
)

type SkillFailure struct{ Skill, Message uint16 }

func DecodeSkillFailure(p []byte) (v SkillFailure, err error) {
	if len(p) != 6 {
		return v, fmt.Errorf("invalid GX skill failure length")
	}
	r := NewReader(p)
	v.Skill, v.Message = r.U16(), r.U16()
	return v, r.Err()
}
