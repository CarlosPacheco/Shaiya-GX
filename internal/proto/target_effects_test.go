package proto

import "testing"

func TestNativeTargetEffectsWire(t *testing.T) {
	w := NewWriter(TargetEffects)
	w.U8(TargetEffectMob)
	w.U32(7)
	w.U8(1)
	w.U16(33)
	w.U8(2)
	w.U32(19)
	v, err := DecodeTargetEffects(w.Bytes())
	if err != nil || v.Kind != 2 || v.ID != 7 || len(v.Effects) != 1 || v.Effects[0].Remaining != 19 {
		t.Fatal(v, err)
	}
	for _, p := range [][]byte{nil, w.Bytes()[:10], append(w.Bytes(), 0)} {
		if _, err := DecodeTargetEffects(p); err == nil {
			t.Fatal("malformed snapshot accepted")
		}
	}
	q := TargetEffectsQuery(2, 7)
	if len(q) != 7 || Op(q) != GXTargetEffectsQuery {
		t.Fatal(q)
	}
}
