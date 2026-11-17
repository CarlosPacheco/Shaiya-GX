package proto

import (
	"encoding/binary"
	"fmt"
)

// GXEnchantRules publishes server INI bonuses for authoritative tooltips.
const GXEnchantRules Opcode = 0x7F12

type EnchantRules struct {
	MaxLevel                uint8
	WeaponBonus, ArmorBonus [50]uint32
}

func EnchantRulesPacket(r EnchantRules) []byte {
	w := NewWriter(GXEnchantRules)
	w.U8(r.MaxLevel)
	for _, table := range [2][50]uint32{r.WeaponBonus, r.ArmorBonus} {
		for _, v := range table {
			w.U32(v)
		}
	}
	return w.Bytes()
}

func DecodeEnchantRules(p []byte) (EnchantRules, error) {
	var r EnchantRules
	if len(p) != 403 || Op(p) != GXEnchantRules || p[2] > 49 {
		return r, fmt.Errorf("invalid enchant rules")
	}
	r.MaxLevel = p[2]
	off := 3
	for _, table := range []*[50]uint32{&r.WeaponBonus, &r.ArmorBonus} {
		for i := range table {
			v := binary.LittleEndian.Uint32(p[off:])
			off += 4
			if v > 2147483647 {
				return EnchantRules{}, fmt.Errorf("invalid enchant bonus")
			}
			table[i] = v
		}
	}
	return r, nil
}
