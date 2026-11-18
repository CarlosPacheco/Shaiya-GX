package proto

import "fmt"

// GXTransform tells the users around a character that it looks like a monster
// (Degeneration) or is back to normal: character id uint32 and the monster type
// uint16, 0 when the transformation ends. The server also repeats it for every
// transformed character that enters a user's view.
const GXTransform Opcode = 0x7F18

type Transform struct {
	User uint32
	Mob  uint16
}

func DecodeTransform(p []byte) (v Transform, err error) {
	if len(p) != 8 {
		return v, fmt.Errorf("invalid GX transform length")
	}
	r := NewReader(p)
	v.User, v.Mob = r.U32(), r.U16()
	return v, r.Err()
}
