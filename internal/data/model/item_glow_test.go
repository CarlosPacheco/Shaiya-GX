package model

import (
	"encoding/binary"
	"testing"
)

func TestITMGlowAndMaterialAreIndependent(t *testing.T) {
	for _, format := range []int32{0, 1} {
		for glow := int32(-1); glow <= 4; glow++ {
			for _, flags := range []int32{0, 1} {
				b := append([]byte("ITM"), make([]byte, 12)...)
				binary.LittleEndian.PutUint32(b[11:], 1)
				words := []int32{0, 0, glow, 22, format}
				if format == 1 {
					words = append(words, 3, -1, 0, 0x3f800000)
				}
				words = append(words, flags)
				for _, v := range words {
					b = binary.LittleEndian.AppendUint32(b, uint32(v))
				}
				m, err := ParseITM(b)
				if err != nil {
					t.Fatal(err)
				}
				r := m.Records[0]
				if r.Glow != glow || r.AlphaTest() != (flags == 0) {
					t.Fatalf("format%d glow%d flags%d: %+v", format, glow, flags, r)
				}
			}
		}
	}
}
