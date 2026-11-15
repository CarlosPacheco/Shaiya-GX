package game

import (
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"testing"
)

func TestOJLimitChunksAndReplacement(t *testing.T) {
	st := New()
	for _, limits := range [][]proto.OJLimit{{{Type: 16, TypeID: 55, Stats: 3, Value: 13}}, {{Type: 1, TypeID: 1, Stats: 4, Value: 11}}, {{Type: 16, TypeID: 55}}} {
		if _, err := st.Apply(proto.OJLimitsPacket(limits)); err != nil {
			t.Fatal(err)
		}
	}
	if len(st.OJLimits) != 2 || st.OJLimits[[2]uint8{16, 55}].Stats != 0 || st.OJLimits[[2]uint8{1, 1}].Stats != 4 {
		t.Fatal(st.OJLimits)
	}
	before := len(st.OJLimits)
	if _, err := st.Apply(proto.OJLimitsPacket([]proto.OJLimit{{Type: 2, TypeID: 1, Stats: 3, Value: 13}, {Type: 1, TypeID: 1, Stats: 10, Value: 13}})); err == nil {
		t.Fatal("malformed catalog accepted")
	}
	if len(st.OJLimits) != before {
		t.Fatal("invalid chunk partially applied")
	}
}
