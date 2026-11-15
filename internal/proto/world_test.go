package proto

import "testing"

func TestGXMobCountryMetadataAndLegacyHeight(t *testing.T) {
	for _, country := range []uint8{0, 1, 2} {
		for _, height := range []uint8{0, 1} {
			w := NewWriter(GXMobEnter)
			w.U32(9)
			w.Bool(true)
			w.U16(252)
			w.F32(10)
			w.F32(20)
			w.F32(30)
			legacy := append([]byte(nil), w.Bytes()...)
			w.U8(country)
			w.U8(height)
			m, err := DecodeMobEnter(w.Bytes())
			if err != nil || !m.CountryKnown || m.Country != country || m.HeightKnown != (height == 1) || m.Y != 20 {
				t.Fatalf("metadata %+v: %v", m, err)
			}
			old, err := DecodeMobEnter(legacy)
			if err != nil || old.CountryKnown || !old.HeightKnown {
				t.Fatalf("legacy %+v: %v", old, err)
			}
			bad := append(w.Bytes(), 0)
			if _, err := DecodeMobEnter(bad); err == nil {
				t.Fatal("trailing metadata accepted")
			}
		}
	}
	for _, pair := range [][2]uint8{{3, 1}, {0, 2}} {
		w := NewWriter(GXMobEnter)
		w.U32(9)
		w.Bool(false)
		w.U16(52)
		w.F32(0)
		w.F32(0)
		w.F32(0)
		w.U8(pair[0])
		w.U8(pair[1])
		if _, err := DecodeMobEnter(w.Bytes()); err == nil {
			t.Fatal("invalid metadata accepted", pair)
		}
	}
}
