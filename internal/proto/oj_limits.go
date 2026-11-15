package proto

import (
	"encoding/binary"
	"fmt"
)

// GXOJLimits publishes static equipment recreation limits from the server DB.
const GXOJLimits Opcode = 0x7F0A

type OJLimit struct{ Type, TypeID, Stats, Value uint8 }

func OJLimitsPacket(limits []OJLimit) []byte {
	w := NewWriter(GXOJLimits)
	w.U16(uint16(len(limits)))
	for _, v := range limits {
		w.U8(v.Type)
		w.U8(v.TypeID)
		w.U8(v.Stats)
		w.U8(v.Value)
	}
	return w.Bytes()
}
func DecodeOJLimits(p []byte) ([]OJLimit, error) {
	if len(p) < 4 || Op(p) != GXOJLimits || len(p) != 4+4*int(binary.LittleEndian.Uint16(p[2:4])) {
		return nil, fmt.Errorf("invalid OJ limits")
	}
	out := make([]OJLimit, 0, (len(p)-4)/4)
	for i := 4; i < len(p); i += 4 {
		v := OJLimit{p[i], p[i+1], p[i+2], p[i+3]}
		if v.Type == 0 || v.TypeID == 0 || v.Stats > 9 || v.Value > 99 || (v.Stats == 0) != (v.Value == 0) {
			return nil, fmt.Errorf("invalid OJ limit")
		}
		out = append(out, v)
	}
	return out, nil
}
