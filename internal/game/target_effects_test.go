package game

import (
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"testing"
)

func TestTargetEffectsSnapshotsAndConfirmedRemoval(t *testing.T) {
	s := New()
	w := proto.NewWriter(proto.TargetEffects)
	w.U8(2)
	w.U32(10)
	w.U8(1)
	w.U16(33)
	w.U8(1)
	w.U32(20)
	if _, err := s.Apply(w.Bytes()); err != nil {
		t.Fatal(err)
	}
	for _, id := range []uint32{11, 10} {
		w = proto.NewWriter(proto.TargetEffectRemove)
		w.U8(2)
		w.U32(id)
		w.U16(33)
		w.U8(1)
		if _, err := s.Apply(w.Bytes()); err != nil {
			t.Fatal(err)
		}
		want := 1
		if id == 10 {
			want = 0
		}
		if len(s.TargetEffects.Effects) != want {
			t.Fatal("removal applied to wrong target", id, s.TargetEffects)
		}
	}
}
