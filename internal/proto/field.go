package proto

import "fmt"

// GXField announces a field left on the ground (Fire Bush's curtain of fire) to
// the users around it: owner uint32, skill uint16, level uint8, position
// x, y, z float32 and the seconds it lasts uint16. The client shows the field's
// area effect there until it ends. Traps are not announced: they are invisible.
const GXField Opcode = 0x7F17

type Field struct {
	Owner   uint32
	Skill   uint16
	Level   uint8
	X, Y, Z float32
	Seconds uint16
}

func DecodeField(p []byte) (v Field, err error) {
	if len(p) != 23 {
		return v, fmt.Errorf("invalid GX field length")
	}
	r := NewReader(p)
	v.Owner, v.Skill, v.Level = r.U32(), r.U16(), r.U8()
	v.X, v.Y, v.Z = r.F32(), r.F32(), r.F32()
	v.Seconds = r.U16()
	return v, r.Err()
}
