//go:build integration

package app

import (
	"github.com/Spelunkern/Shaiya-GX/internal/game"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/testworld"
	"testing"
)

func TestTargetEffectsAgainstIsolatedGX(t *testing.T) {
	st, _, send, wait := testworld.Enter(t)
	send(proto.TargetEffectsQuery(proto.TargetEffectUser, st.SelfID))
	wait(func() bool {
		return st.TargetEffects.Kind == proto.TargetEffectUser && st.TargetEffects.ID == st.SelfID && len(st.TargetEffects.Effects) == len(st.Buffs)
	})
	var mob uint32
	t.Logf("query fixture map=%d position=(%.2f,%.2f) entities=%d", st.MapID, st.Self.X, st.Self.Z, len(st.Entities))
	wait(func() bool {
		for id, e := range st.Entities {
			if e.Kind == game.KindMob {
				mob = id
				return true
			}
		}
		return false
	})
	send(proto.TargetEffectsQuery(proto.TargetEffectMob, mob))
	wait(func() bool { return st.TargetEffects.Kind == proto.TargetEffectMob && st.TargetEffects.ID == mob })
	if st.Unknown[proto.TargetEffects] != 0 {
		t.Fatal("native target-effect snapshot remained unknown")
	}
}
