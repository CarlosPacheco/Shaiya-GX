package proto

import "fmt"

// GXLadder is additive. Requests contain guide index + mode; notifications also
// identify the actor. Native movement positions and flags retain their layouts.
const GXLadder Opcode = 0x7F0D

type LadderState struct {
	ID    uint32
	Index uint16
	Mode  uint8
}

func LadderPacket(index uint16, mode uint8) []byte {
	w := NewWriter(GXLadder)
	w.U16(index)
	w.U8(mode)
	return w.Bytes()
}
func DecodeLadderState(p []byte) (v LadderState, err error) {
	if len(p) != 9 {
		return v, fmt.Errorf("ladder state: expected 9 bytes, got %d", len(p))
	}
	r := NewReader(p)
	v.ID = r.U32()
	v.Index = r.U16()
	v.Mode = r.U8()
	if v.Mode > 3 {
		return v, fmt.Errorf("invalid ladder mode")
	}
	return v, r.Err()
}
