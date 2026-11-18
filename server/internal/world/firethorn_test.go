package world

import (
	"testing"
	"time"

	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/server/internal/db"
)

// Fire Thorn (TypeEffect 4, TypeDetail 84): every three seconds hostile mobs
// and users within ApplyRange take the skill's DamageHP as fixed damage.
func TestFireThornPulsesFixedDamageOnMobsEveryThreeSeconds(t *testing.T) {
	s, defs := lapisSession(t)
	id, _ := lapisMob(t, s, defs)
	s.player = RegisterPlayer(*s.char, nil, false, s.faction, s.out, s.logger)
	t.Cleanup(s.player.Unregister)
	m, _ := MobByID(s.char.MapID, id)
	s.char.PosX, s.char.PosY, s.char.PosZ = m.PosX, m.PosY, m.PosZ
	s.skillCooldowns = map[uint16]time.Time{}
	def := defs.Skill(47, 3)
	if def == nil || def.TypeDetail != detailFireThorn || def.DamageHP != 65 || def.ApplyRange != 2 {
		t.Fatalf("Fire Thorn 3 definition: %+v", def)
	}
	if _, ok := s.addEffect(def, 0, s.char.ID); !ok || !s.hasBuff(47) {
		t.Fatal("the buff was not applied")
	}
	before, _ := MobByID(s.char.MapID, id)
	now := time.Now()

	s.tickFireThorn(now.Add(time.Second))
	if mid, _ := MobByID(s.char.MapID, id); mid.CurrentHP != before.CurrentHP {
		t.Fatal("a pulse fired before three seconds")
	}
	s.tickFireThorn(now.Add(fireThornTick + time.Second))
	first, _ := MobByID(s.char.MapID, id)
	if before.CurrentHP-first.CurrentHP != 65 {
		t.Fatalf("pulse damage %d, want the skill's fixed 65", before.CurrentHP-first.CurrentHP)
	}
	s.tickFireThorn(now.Add(fireThornTick + 2*time.Second))
	if again, _ := MobByID(s.char.MapID, id); again.CurrentHP != first.CurrentHP {
		t.Fatal("pulses must be three seconds apart")
	}
	s.tickFireThorn(now.Add(2*fireThornTick + 2*time.Second))
	if next, _ := MobByID(s.char.MapID, id); first.CurrentHP-next.CurrentHP != 65 {
		t.Fatal("the next pulse did not hit")
	}

	// A mob out of reach is untouched.
	s.char.PosX += 50
	hp, _ := MobByID(s.char.MapID, id)
	s.tickFireThorn(now.Add(3*fireThornTick + 3*time.Second))
	if far, _ := MobByID(s.char.MapID, id); far.CurrentHP != hp.CurrentHP {
		t.Fatal("a mob outside ApplyRange was hurt")
	}
}

func TestFireThornHurtsHostileUsersIgnoringStats(t *testing.T) {
	s, wire := potentialSession(t)
	def := db.MustDefs().Skill(47, 1)
	s.skills = []db.CharacterSkill{{SkillID: 47, Level: 1, Number: 1}}
	if _, ok := s.addEffect(def, 0, s.char.ID); !ok {
		t.Fatal("the buff was not applied")
	}
	// A hostile user next to the caster: the other country.
	other, _ := gmWireSession(t, *s.char, false)
	other.faction = FactionDark
	if s.faction == FactionDark {
		other.faction = FactionLight
	}
	other.char.ID, other.char.Name = s.char.ID+500, "ThornVictim"
	other.skillCooldowns, other.itemCooldowns = map[uint16]time.Time{}, map[uint8]time.Time{}
	other.player = RegisterPlayer(*other.char, nil, false, other.faction, other.out, other.logger)
	t.Cleanup(other.player.Unregister)
	other.char.CurrentHP = other.stats().MaxHP
	hp := other.stats().CurrentHP
	other.player.mu.Lock()
	other.player.character.PosX, other.player.character.PosZ = s.char.PosX+1, s.char.PosZ
	other.player.mu.Unlock()

	s.tickFireThorn(time.Now().Add(fireThornTick + time.Second))
	other.tickUserHits()
	if lost := hp - other.stats().CurrentHP; lost != 8 {
		t.Fatalf("the user lost %d HP, want the fixed 8", lost)
	}
	seen := false
	for len(wire) > 0 {
		if p := <-wire; proto.Op(p) == proto.SkillKeep {
			seen = true
		}
	}
	if !seen {
		t.Fatal("nearby users did not get the periodic damage packet")
	}
}
