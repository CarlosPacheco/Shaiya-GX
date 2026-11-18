package proto

import "fmt"

// GXTargetHealth gives the HP of the user the client has selected (the native
// protocol only tells the HP of party members): character id, current and
// maximum HP, all uint32. The server repeats it whenever it changes.
const GXTargetHealth Opcode = 0x7F19

type TargetHealth struct {
	User    uint32
	HP, Max uint32
}

func DecodeTargetHealth(p []byte) (v TargetHealth, err error) {
	if len(p) != 14 {
		return v, fmt.Errorf("invalid GX target health length")
	}
	r := NewReader(p)
	v.User, v.HP, v.Max = r.U32(), r.U32(), r.U32()
	return v, r.Err()
}
