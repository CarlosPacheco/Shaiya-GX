package world

import "testing"

// Taunt (TypeEffect 4, TypeDetail 69) is pure aggro: every hostile mob within
// ApplyRange turns to the caster and its ledger gains Arrow plus ten times the
// caster's REC, with no damage.
func TestTauntAddsOnlyAggroToMobsInRange(t *testing.T) {
	s, defs := lapisSession(t)
	id, _ := lapisMob(t, s, defs)
	m, _ := MobByID(s.char.MapID, id)
	s.char.PosX, s.char.PosY, s.char.PosZ = m.PosX, m.PosY, m.PosZ
	def := defs.Skill(105, 3)
	if def == nil || def.TypeDetail != detailTaunt || def.Aggro != 2000 {
		t.Fatalf("Taunt 3 definition: %+v", def)
	}
	stats := s.stats()
	before, _ := MobByID(s.char.MapID, id)

	s.tauntMobs(def, &stats)
	after, _ := MobByID(s.char.MapID, id)
	if after.CurrentHP != before.CurrentHP {
		t.Fatal("Taunt must not deal damage")
	}
	if after.AggroTarget != s.char.ID {
		t.Fatalf("the mob did not turn to the caster: %d", after.AggroTarget)
	}
	got := mobAggro(t, s.char.MapID, id, s.char.ID)
	if want := int64(stats.Rec)*10 + 2000; got != want {
		t.Fatalf("aggro %d, want Arrow + 10 x REC = %d", got, want)
	}

	// Out of reach: nothing happens.
	s.char.PosX += 50
	s.tauntMobs(def, &stats)
	if again := mobAggro(t, s.char.MapID, id, s.char.ID); again != got {
		t.Fatal("a mob outside ApplyRange gained aggro")
	}
}

func TestHitAggroIgnoresDamage(t *testing.T) {
	stats := CharacterStats{Rec: 12}
	if hitAggro(&stats, 0) != 120 || hitAggro(&stats, 300) != 420 {
		t.Fatal("aggro is REC x 10 plus the skill's Arrow")
	}
}

func mobAggro(t *testing.T, mapID uint16, mobID, characterID uint32) int64 {
	t.Helper()
	sh := shard(mapID)
	sh.mu.Lock()
	defer sh.mu.Unlock()
	for i := range sh.entities.Mobs {
		if m := &sh.entities.Mobs[i]; m.ID == mobID {
			return m.aggro[characterID]
		}
	}
	t.Fatal("mob not found")
	return 0
}
