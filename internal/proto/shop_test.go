package proto

import "testing"

func TestItemDropPacketIsTheNativeRemoveRequest(t *testing.T) {
	p := ItemDropPacket(2, 5, 7)
	if Op(p) != RemoveItem || p[2] != 2 || p[3] != 5 {
		t.Fatalf("% x", p)
	}
}
