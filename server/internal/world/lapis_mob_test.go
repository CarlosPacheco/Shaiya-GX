package world

import (
	"github.com/Spelunkern/Shaiya-GX/server/internal/db"
	"testing"
	"time"
)

func lapisMob(t *testing.T, s *Session, defs *db.Defs) (uint32, *db.MobDef) {
	t.Helper()
	s.char.MapID = 61990
	md := &db.MobDef{MobID: 60001, Level: 1, MaxHP: 10000, MaxSP: 10000, MaxMP: 10000, Country: 2}
	defs.Mobs[md.MobID] = md
	m := NewStaticMob(md, md.MobID, 1, 1, nil, 0)
	sh := shard(s.char.MapID)
	sh.mu.Lock()
	old := sh.entities
	sh.entities = MapEntities{Mobs: []MapMob{m}}
	sh.mu.Unlock()
	t.Cleanup(func() { sh.mu.Lock(); sh.entities = old; sh.mu.Unlock() })
	return m.ID, md
}

func TestGemNPCProcAppliesToMobHonorsChanceAndPreventsRecursion(t *testing.T) {
	s, defs := lapisSession(t)
	id, md := lapisMob(t, s, defs)
	npc := *defs.NPCSkill(205)
	npc.SuccessValue = 0
	defs.NPCSkills[db.SkillKey{SkillID: 205, Level: 1}] = &npc
	s.items[0].Gems[0] = 151
	s.gemCastsOnMob(id, weaponGemSkills(s.items))
	m, _ := MobByID(s.char.MapID, id)
	if len(m.Effects) != 0 {
		t.Fatal("0% proc applied")
	}
	npc.SuccessValue = 100
	md.ResistState[1] = 1
	s.gemCastsOnMob(id, weaponGemSkills(s.items))
	m, _ = MobByID(s.char.MapID, id)
	if len(m.Effects) != 0 {
		t.Fatal("state immunity ignored")
	}
	md.ResistState[1] = 0
	s.gemCastsOnMob(id, weaponGemSkills(s.items))
	m, _ = MobByID(s.char.MapID, id)
	if len(m.Effects) != 1 || m.mods().canAct() || m.Effects[0].Def.SkillLevel != 100 || s.itemAttacking {
		t.Fatal("proc did not apply once or recursion guard leaked", m.Effects)
	}
}

func TestLapisPeriodicResourceDamageLeavesMobHPAndSleepAlone(t *testing.T) {
	s, defs := lapisSession(t)
	id, _ := lapisMob(t, s, defs)
	for _, gem := range []uint8{160, 161} {
		d := defs.Item(30, gem)
		sk := *defs.NPCSkill(uint16(d.Special))
		sk.SuccessValue = 100
		defs.NPCSkills[db.SkillKey{SkillID: sk.SkillID, Level: 1}] = &sk
		s.items[0].Gems[0] = gem
		s.gemCastsOnMob(id, weaponGemSkills(s.items))
	}
	now := time.Now()
	sh := shard(s.char.MapID)
	sh.mu.Lock()
	for i := range sh.entities.Mobs[0].Effects {
		sh.entities.Mobs[0].Effects[i].prevTick = now.Add(-effectTick)
	}
	sh.mu.Unlock()
	outcomes := tickMobEffects(now)
	m, _ := MobByID(s.char.MapID, id)
	if len(m.Effects) != 2 || m.CurrentHP != 10000 {
		t.Fatalf("resource skills should apply without damaging the mob: hp=%d effects=%+v", m.CurrentHP, m.Effects)
	}
	for _, outcome := range outcomes {
		if outcome.Mob.ID == id && outcome.Damaged {
			t.Fatalf("resource tick sent monster damage: %+v", outcome.Damage)
		}
	}
	// Restrictions on mob categories use positive disable counters too.
	for _, skill := range []uint16{211, 214} {
		ApplyMobEffect(s.char.MapID, id, s.char.ID, defs.NPCSkill(skill))
	}
	m, _ = MobByID(s.char.MapID, id)
	if m.mods().Disabled[0] <= 0 || m.mods().Disabled[2] <= 0 {
		t.Fatal("mob blindness/silence did not disable categories")
	}
	ApplyMobEffect(s.char.MapID, id, s.char.ID, defs.NPCSkill(241))
	if _, ok := DamageMobFrom(s.char.MapID, id, 0, s.char.ID); !ok {
		t.Fatal("zero hit failed")
	}
	m, _ = MobByID(s.char.MapID, id)
	if m.mods().Sleep == 0 {
		t.Fatal("zero HP damage woke mob")
	}
	DamageMobFrom(s.char.MapID, id, 1, s.char.ID)
	m, _ = MobByID(s.char.MapID, id)
	if m.mods().Sleep != 0 {
		t.Fatal("HP damage did not wake mob")
	}
}
