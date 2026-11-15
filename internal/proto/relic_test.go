package proto

import (
	"math"
	"testing"
)

func TestRelicRuntimeIDsPortalStatesAndMalformedSnapshots(t *testing.T) {
	list := func(id uint32, owner uint8, x float32) []byte {
		w := NewWriter(RelicList)
		w.U8(1)
		w.U32(id)
		w.U8(owner)
		w.F32(x)
		w.F32(200)
		return w.Bytes()
	}
	for _, owner := range []uint8{0, 1, 2, 100, 101, 102, 103} {
		v, err := DecodeRelicList(list(0, owner, 100))
		if err != nil || len(v) != 1 || v[0].ID != 0 || v[0].Owner != owner {
			t.Fatalf("runtime ID zero/state %d: %+v %v", owner, v, err)
		}
	}
	for _, p := range [][]byte{list(0, 3, 100), list(1, 1, float32(math.NaN())), append(list(0, 0, 100), 0), {1, 0x21, 101}} {
		if _, err := DecodeRelicList(p); err == nil {
			t.Fatalf("accepted malformed snapshot %x", p)
		}
	}
	w := NewWriter(PortalNotice)
	w.U16(1060)
	w.U8(100)
	w.U8(101)
	if v, err := DecodePortalNotice(w.Bytes()); err != nil || v.PortalID != 1060 || v.NewOwner != 101 {
		t.Fatal(v, err)
	}
}
