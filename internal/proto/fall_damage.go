package proto

import "fmt"

// GXFallDamage is an additive server notification. Health still uses 0x0521.
// No client-authored fall height or damage report is accepted by the server.
const GXFallDamage Opcode = 0x7F0C

type EnvironmentalDamage struct{ Target, Amount uint32 }

type FallDamage = EnvironmentalDamage

func DecodeFallDamage(p []byte) (FallDamage, error) {
	return decodeEnvironmentalDamage(p, "fall")
}

func decodeEnvironmentalDamage(p []byte, cause string) (v EnvironmentalDamage, err error) {
	if len(p) != 10 {
		return v, fmt.Errorf("%s damage: expected 10 bytes, got %d", cause, len(p))
	}
	r := NewReader(p)
	v.Target, v.Amount = r.U32(), r.U32()
	return v, r.Err()
}
