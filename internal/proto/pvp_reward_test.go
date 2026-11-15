package proto

import "testing"

func TestPvPRewardNativeLayouts(t *testing.T) {
	w := NewWriter(PvPRewardStatus)
	w.U8(1)
	w.U8(32)
	if v, err := DecodePvPRewards(w.Bytes()); err != nil || v.KillStage != 1 || v.DeathStage != 32 || !v.Known {
		t.Fatal(v, err)
	}
	for _, death := range []bool{false, true} {
		op := PvPRewardKill
		if death {
			op = PvPRewardDeath
		}
		w = NewWriter(op)
		w.U8(1)
		if death {
			w.U32(1000000)
		} else {
			w.U8(28)
		}
		p := w.Bytes()
		v, err := DecodePvPRewardResult(p, death)
		if err != nil || !v.OK || (death && v.Value != 1000000) || (!death && v.Value != 28) {
			t.Fatal(v, err)
		}
		for n := 2; n < len(p); n++ {
			if _, err = DecodePvPRewardResult(p[:n], death); err == nil {
				t.Fatal("truncated reward accepted")
			}
		}
		if _, err = DecodePvPRewardResult(append(p, 0), death); err == nil {
			t.Fatal("trailing reward data accepted")
		}
	}
}
