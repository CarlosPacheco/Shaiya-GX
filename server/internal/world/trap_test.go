package world

import (
	"testing"
	"time"

	"github.com/Spelunkern/Shaiya-GX/server/internal/db"
)

func TestTrapDamageFollowsTrabDamage(t *testing.T) {
	cur, max := [3]uint32{1000, 400, 300}, [3]uint32{2000, 500, 400}
	cases := []struct {
		def  db.SkillDef
		want [3]uint32
	}{
		{db.SkillDef{DamageType: 5, DamageHP: 15}, [3]uint32{150, 0, 0}},                              // % of current
		{db.SkillDef{DamageType: 4, DamageHP: 10, DamageSP: 50, DamageMP: 1}, [3]uint32{200, 250, 4}}, // % of maximum
		{db.SkillDef{DamageType: 5, DamageHP: 1}, [3]uint32{10, 0, 0}},
		{db.SkillDef{DamageType: 0, DamageHP: 77, DamageMP: 9}, [3]uint32{77, 0, 9}}, // fixed
		{db.SkillDef{DamageType: 4, DamageHP: 100}, [3]uint32{2000, 0, 0}},
	}
	for i, c := range cases {
		if got := trapDamage(&c.def, cur, max); got != c.want {
			t.Fatalf("case %d: %v, want %v", i, got, c.want)
		}
	}
	// A tiny percentage still hurts, and the cap is 30000.
	if got := trapDamage(&db.SkillDef{DamageType: 5, DamageHP: 1}, [3]uint32{5}, [3]uint32{5}); got[0] != 1 {
		t.Fatalf("minimum %v", got)
	}
	if got := trapDamage(&db.SkillDef{DamageType: 4, DamageHP: 100}, [3]uint32{}, [3]uint32{90000}); got[0] != 30000 {
		t.Fatalf("cap %v", got)
	}
}

func trapSession(t *testing.T) (*Session, *db.SkillDef, *db.Defs) {
	s, defs := lapisSession(t)
	s.skillCooldowns = map[uint16]time.Time{}
	s.player = RegisterPlayer(*s.char, nil, false, s.faction, s.out, s.logger)
	t.Cleanup(s.player.Unregister)
	def := defs.Skill(113, 1)
	if def == nil || def.TypeEffect != typeEffectField || def.TypeDetail != detailTrap || def.ApplyRange != 2 || def.KeepSeconds != 30 {
		t.Fatalf("Trap 1 definition: %+v", def)
	}
	return s, def, defs
}

// A trap is placed where the caster stands, pays the skill's costs and starts
// its reuse; there is no limit of traps.
func TestTrapIsPlacedWithoutLimit(t *testing.T) {
	s, def, _ := trapSession(t)
	k := db.CharacterSkill{SkillID: def.SkillID, Level: def.SkillLevel, Number: 1}
	st := s.stats()
	s.char.CurrentSP, s.char.CurrentMP = st.MaxSP, st.MaxMP
	for range 3 {
		s.skillField(k, def, s.stats())
	}
	if len(s.traps) != 3 {
		t.Fatalf("%d traps placed, want 3", len(s.traps))
	}
	if s.traps[0].x != s.char.PosX || time.Until(s.traps[0].endsAt) > 31*time.Second || time.Until(s.traps[0].endsAt) < 29*time.Second {
		t.Fatalf("trap: %+v", s.traps[0])
	}
}

func TestTrapHurtsTheFirstEnemyMobOnceAndIsConsumed(t *testing.T) {
	s, def, defs := trapSession(t)
	id, _ := lapisMob(t, s, defs)
	m, _ := MobByID(s.char.MapID, id)
	s.char.PosX, s.char.PosY, s.char.PosZ = m.PosX+1, m.PosY, m.PosZ
	s.traps = []placedTrap{{mapID: s.char.MapID, x: m.PosX, y: m.PosY, z: m.PosZ, def: def, endsAt: time.Now().Add(time.Minute)}}
	before, _ := MobByID(s.char.MapID, id)
	s.tickTraps(time.Now())
	after, _ := MobByID(s.char.MapID, id)
	want := uint32(float64(def.DamageHP) / 100 * float64(before.CurrentHP))
	if before.CurrentHP-after.CurrentHP != want || want == 0 {
		t.Fatalf("the mob lost %d, want %d (%d%% of its current HP)", before.CurrentHP-after.CurrentHP, want, def.DamageHP)
	}
	if len(s.traps) != 0 {
		t.Fatal("a sprung trap must be gone")
	}
	s.tickTraps(time.Now().Add(time.Second))
	if again, _ := MobByID(s.char.MapID, id); again.CurrentHP != after.CurrentHP {
		t.Fatal("the trap hit twice")
	}
}

func TestTrapIgnoresFarMobsAndExpires(t *testing.T) {
	s, def, defs := trapSession(t)
	id, _ := lapisMob(t, s, defs)
	m, _ := MobByID(s.char.MapID, id)
	s.char.PosX, s.char.PosY, s.char.PosZ = m.PosX, m.PosY, m.PosZ
	s.traps = []placedTrap{{mapID: s.char.MapID, x: m.PosX + 30, y: m.PosY, z: m.PosZ, def: def, endsAt: time.Now().Add(time.Minute)}}
	s.tickTraps(time.Now())
	if len(s.traps) != 1 {
		t.Fatal("a trap out of reach sprang")
	}
	s.tickTraps(time.Now().Add(2 * time.Minute))
	if len(s.traps) != 0 {
		t.Fatal("an expired trap stayed")
	}
}

func TestTrapSparesImmuneMobs(t *testing.T) {
	s, def, defs := trapSession(t)
	id, _ := lapisMob(t, s, defs)
	m, _ := MobByID(s.char.MapID, id)
	md := db.MustDefs().Mobs[m.MobID]
	md.ResistState[def.StateType-1] = 1
	t.Cleanup(func() { md.ResistState[def.StateType-1] = 0 })
	s.traps = []placedTrap{{mapID: s.char.MapID, x: m.PosX, y: m.PosY, z: m.PosZ, def: def, endsAt: time.Now().Add(time.Minute)}}
	s.tickTraps(time.Now())
	if len(s.traps) != 1 {
		t.Fatal("a trap sprang on an immune mob")
	}
}

func TestTrapHurtsAnEnemyUserAndRevealsIt(t *testing.T) {
	s, def, _ := trapSession(t)
	foe := detectionUser(t, s, 1, true)
	ally := detectionUser(t, s, 1, false)
	for _, u := range []*Session{foe, ally} {
		if _, ok := u.addEffect(db.MustDefs().Skill(63, 1), 0, u.char.ID); !ok {
			t.Fatal("stealth was not applied")
		}
		st := u.stats()
		u.char.CurrentHP = st.MaxHP
	}
	hp := foe.stats().CurrentHP
	allyHP := ally.stats().CurrentHP
	s.traps = []placedTrap{{mapID: s.char.MapID, x: s.char.PosX + 1, y: s.char.PosY, z: s.char.PosZ, def: def, endsAt: time.Now().Add(time.Minute)}}
	s.tickTraps(time.Now())
	foe.tickUserHits()
	ally.tickUserHits()
	want := uint32(float64(def.DamageHP) / 100 * float64(hp))
	if lost := hp - foe.stats().CurrentHP; lost != want || want == 0 {
		t.Fatalf("the enemy lost %d, want %d", lost, want)
	}
	if foe.player.hidden.Load() {
		t.Fatal("a sprung trap must end the victim's stealth")
	}
	if ally.stats().CurrentHP != allyHP || !ally.player.hidden.Load() {
		t.Fatal("an ally was hurt by the trap")
	}
}
