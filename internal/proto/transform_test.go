package proto

import "testing"

func TestTransformRoundTrip(t *testing.T) {
	w := NewWriter(GXTransform)
	w.U32(9)
	w.U16(60)
	v, err := DecodeTransform(w.Bytes())
	if err != nil || v != (Transform{User: 9, Mob: 60}) {
		t.Fatalf("%+v %v", v, err)
	}
	if _, err := DecodeTransform(w.Bytes()[:6]); err == nil {
		t.Fatal("short packet accepted")
	}
}
