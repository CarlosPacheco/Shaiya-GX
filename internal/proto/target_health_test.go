package proto

import "testing"

func TestTargetHealthRoundTrip(t *testing.T) {
	w := NewWriter(GXTargetHealth)
	w.U32(7)
	w.U32(1234)
	w.U32(5000)
	v, err := DecodeTargetHealth(w.Bytes())
	if err != nil || v != (TargetHealth{User: 7, HP: 1234, Max: 5000}) {
		t.Fatalf("%+v %v", v, err)
	}
	if _, err := DecodeTargetHealth(w.Bytes()[:10]); err == nil {
		t.Fatal("short packet accepted")
	}
}
