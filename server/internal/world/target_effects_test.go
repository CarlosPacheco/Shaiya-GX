package world

import (
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/server/internal/db"
	"testing"
	"time"
)

func TestTargetEffectSnapshotIsVisibleAuthoritativeAndClearsOnRemoval(t *testing.T) {
	s, wire := gmWireSession(t, db.Character{ID: 67001, MapID: 60099}, false)
	s.player = RegisterPlayer(*s.char, nil, false, 0, s.out, s.logger)
	defer s.player.Unregister()
	d := &db.SkillDef{SkillID: 33, SkillLevel: 1, KeepSeconds: 30}
	now := time.Now()
	sh := shard(s.char.MapID)
	sh.mu.Lock()
	sh.entities = MapEntities{Mobs: []MapMob{{ID: 67002, CurrentHP: 100, Effects: []MobEffect{{Def: d, endsAt: now.Add(20 * time.Second)}}}}}
	sh.mu.Unlock()
	t.Cleanup(func() { sh.mu.Lock(); sh.entities = MapEntities{}; sh.mu.Unlock() })
	read := func() proto.TargetEffectSnapshot {
		t.Helper()
		select {
		case p := <-wire:
			v, err := proto.DecodeTargetEffects(p)
			if err != nil {
				t.Fatal(err)
			}
			return v
		case <-time.After(time.Second):
			t.Fatal("no native effect snapshot")
			return proto.TargetEffectSnapshot{}
		}
	}
	s.handleTargetEffects(proto.TargetEffectsQuery(2, 67002))
	if len(read().Effects) != 0 || s.effectWatchID != 0 {
		t.Fatal("hidden mob effects leaked")
	}
	s.player.know(viewMob, 67002)
	s.handleTargetEffects(proto.TargetEffectsQuery(2, 67002))
	v := read()
	if len(v.Effects) != 1 || v.Effects[0].Skill != 33 || v.Effects[0].Remaining > 20 {
		t.Fatal(v)
	}
	sh.mu.Lock()
	sh.entities.Mobs[0].Effects = nil
	sh.mu.Unlock()
	s.tickTargetEffects(now.Add(time.Second))
	if len(read().Effects) != 0 {
		t.Fatal("cured effect remained visible")
	}
}
