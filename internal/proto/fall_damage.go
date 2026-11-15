package proto

import "fmt"

// GXFallDamage is an additive server notification. Health still uses 0x0521.
// No client-authored fall height or damage report is accepted by the server.
const GXFallDamage Opcode = 0x7F0C

type FallDamage struct{ Target, Amount uint32 }

func DecodeFallDamage(p []byte) (v FallDamage, err error) {
	if len(p) != 10 {
		return v, fmt.Errorf("fall damage: expected 10 bytes, got %d", len(p))
	}
	r := NewReader(p)
	v.Target, v.Amount = r.U32(), r.U32()
	return v, r.Err()
}
