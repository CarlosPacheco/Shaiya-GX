package proto

import "testing"

func TestKillNoticeRoundTrip(t *testing.T) {
	w := NewWriter(GXKillNotice)
	w.Fixed("Victim", 21, false)
	name, err := DecodeKillNotice(w.Bytes())
	if err != nil || name != "Victim" {
		t.Fatalf("%q %v", name, err)
	}
	if _, err := DecodeKillNotice(w.Bytes()[:10]); err == nil {
		t.Fatal("short packet accepted")
	}
}
