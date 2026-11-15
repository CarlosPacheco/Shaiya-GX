package proto

import (
	"encoding/binary"
	"fmt"
)

// GXChestCloakFits supplies the native chest-dependent cloak fit. It is an
// additive cosmetic catalog; existing packet layouts and game rules are unchanged.
const GXChestCloakFits Opcode = 0x7F0B

type ChestCloakFit struct{ Type, TypeID, Fit uint8 }

func ChestCloakFitsPacket(rows []ChestCloakFit) []byte {
	w := NewWriter(GXChestCloakFits)
	w.U16(uint16(len(rows)))
	for _, v := range rows {
		w.U8(v.Type)
		w.U8(v.TypeID)
		w.U8(v.Fit)
	}
	return w.Bytes()
}
func DecodeChestCloakFits(p []byte) ([]ChestCloakFit, error) {
	if len(p) < 4 || Op(p) != GXChestCloakFits || len(p) != 4+3*int(binary.LittleEndian.Uint16(p[2:4])) {
		return nil, fmt.Errorf("invalid chest cloak fits")
	}
	rows := make([]ChestCloakFit, 0, (len(p)-4)/3)
	for i := 4; i < len(p); i += 3 {
		v := ChestCloakFit{p[i], p[i+1], p[i+2]}
		if v.Type == 0 || v.TypeID == 0 || v.Fit > 7 {
			return nil, fmt.Errorf("invalid chest cloak fit")
		}
		rows = append(rows, v)
	}
	return rows, nil
}
