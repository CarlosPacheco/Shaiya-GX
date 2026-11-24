package proto

import "testing"

func TestPortalRequestAndResultCodes(t *testing.T) {
	if got := PortalRequestPacket(4); len(got) != 3 || Op(got) != CharacterEnteredPortal || got[2] != 4 {
		t.Fatalf("request %x", got)
	}
	w := NewWriter(CharacterEnteredPortal)
	w.U8(4)
	w.U8(10)
	r, err := DecodePortalResult(w.Bytes())
	if err != nil || r.Index != 4 || r.Code != 10 {
		t.Fatalf("result %+v %v", r, err)
	}
	// Game.exe 0x569750: party refusals are SysMsg 8042, 6600, 6601 and 6602.
	for code, want := range map[uint8]int{4: 8042, 9: 6600, 10: 6601, 11: 6602, 5: 0, 200: 0} {
		if got := PortalMessageID(code); got != want {
			t.Fatalf("code %d -> %d, want %d", code, got, want)
		}
	}
}
