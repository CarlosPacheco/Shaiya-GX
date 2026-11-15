package proto

import "testing"

func TestLadderRequestAndStateLayouts(t *testing.T) {
	p := LadderPacket(513, 2)
	if len(p) != 5 || Op(p) != GXLadder || p[2] != 1 || p[3] != 2 || p[4] != 2 {
		t.Fatal(p)
	}
	w := NewWriter(GXLadder)
	w.U32(123)
	w.U16(513)
	w.U8(2)
	v, err := DecodeLadderState(w.Bytes())
	if err != nil || v.ID != 123 || v.Index != 513 || v.Mode != 2 {
		t.Fatal(v, err)
	}
	for _, p := range [][]byte{nil, p, append(w.Bytes(), 0)} {
		if _, err := DecodeLadderState(p); err == nil {
			t.Fatal("accepted malformed state")
		}
	}
}
