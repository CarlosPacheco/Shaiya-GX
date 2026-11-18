package proto

import "testing"

func TestFieldRoundTrip(t *testing.T) {
	w := NewWriter(GXField)
	w.U32(7)
	w.U16(48)
	w.U8(3)
	w.F32(1.5)
	w.F32(2.5)
	w.F32(3.5)
	w.U16(12)
	f, err := DecodeField(w.Bytes())
	if err != nil || f != (Field{Owner: 7, Skill: 48, Level: 3, X: 1.5, Y: 2.5, Z: 3.5, Seconds: 12}) {
		t.Fatalf("%+v %v", f, err)
	}
	if _, err := DecodeField(w.Bytes()[:10]); err == nil {
		t.Fatal("a short packet was accepted")
	}
}
