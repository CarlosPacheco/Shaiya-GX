package world

import (
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/server/internal/db"
	"github.com/Spelunkern/Shaiya-GX/server/internal/packet"
)

func TestAttackSacrificeKillsCasterAfterMiss(t *testing.T) {
	for _, tc := range []struct {
		name  string
		level uint8
		gm    bool
	}{
		{"single target", 1, false},
		{"area", 2, false},
		{"area with GM immunity", 3, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s, wire := potentialSession(t)
			s.isAdmin, s.gmImmune = tc.gm, tc.gm
			defs := db.MustDefs()
			k := db.CharacterSkill{SkillID: 87, Level: tc.level, Number: 1}
			catalog := defs.Skill(k.SkillID, k.Level)
			if catalog == nil || catalog.TypeEffect != typeEffectAttack || catalog.TypeDetail != detailSacrifice {
				t.Fatal("fixture needs the catalog's Eraser")
			}
			def := *catalog
			def.SuccessType, def.SuccessValue = 1, 0
			defs.Skills[db.SkillKey{SkillID: k.SkillID, Level: k.Level}] = &def
			s.skills = []db.CharacterSkill{k}

			const mobID = 64913
			defs.Mobs[mobID] = &db.MobDef{MobID: mobID, MaxHP: 1000, Country: 2}
			sh := shard(s.char.MapID)
			sh.mu.Lock()
			sh.entities.Mobs = []MapMob{{ID: mobID, MobID: mobID, PosX: s.char.PosX + 1, PosY: s.char.PosY, PosZ: s.char.PosZ, CurrentHP: 1000, MaxHP: 1000}}
			sh.mu.Unlock()
			t.Cleanup(func() {
				sh.mu.Lock()
				sh.entities.Mobs = nil
				sh.mu.Unlock()
			})

			s.handleUseSkill(packet.SkillUse, proto.UseSkillPacket(k.Number, 0, false))
			if s.dead || !s.skillReady(k.SkillID) {
				t.Fatal("a refused cast killed the caster or started reuse")
			}
			s.handleUseSkill(packet.SkillUse, proto.UseSkillPacket(k.Number, mobID, false))
			if !s.dead || s.char.CurrentHP != 0 || s.player.IsAlive() || s.rebirthAt.IsZero() {
				t.Fatal("accepted sacrificial attack did not leave the caster dead")
			}
			if mob, ok := MobByID(s.char.MapID, mobID); !ok || mob.CurrentHP != 1000 || s.skillReady(k.SkillID) {
				t.Fatal("expected a miss with the accepted skill on cooldown")
			}
			die := proto.NewReader(waitWorldPacket(t, wire, proto.CharacterDeath))
			if die.U32() != s.char.ID {
				t.Fatal("the caster did not receive its death")
			}
		})
	}
}

// Sacrifice (TypeEffect 2, TypeDetail 43): the caster pays its whole SP/MP pool
// and the target recovers the caster's maximum SP (level 1), SP and MP (2) or
// HP (level 3).
func TestSacrificeTransfersTheCastersMaximumPool(t *testing.T) {
	for level := uint8(1); level <= 3; level++ {
		t.Run(string('0'+level), func(t *testing.T) { sacrificeLevel(t, level) })
	}
}

func sacrificeLevel(t *testing.T, level uint8) {
	s, _ := potentialSession(t)
	def := db.MustDefs().Skill(106, level)
	if def == nil || def.TypeDetail != detailSacrificeHeal {
		t.Fatalf("Sacrifice %d missing: %+v", level, def)
	}
	k := db.CharacterSkill{SkillID: 106, Level: level, Number: 1}
	s.skills = []db.CharacterSkill{k}
	st := s.stats()
	s.char.CurrentHP, s.char.CurrentSP, s.char.CurrentMP = st.MaxHP/2, st.MaxSP/2, st.MaxMP/2
	before := s.stats()
	s.friendlySkill(k, def, s.char.ID, before)
	after := s.stats()
	switch level {
	case 1:
		if after.CurrentSP != st.MaxSP || after.CurrentMP != before.CurrentMP || after.CurrentHP != before.CurrentHP {
			t.Fatalf("level 1: SP %d MP %d HP %d", after.CurrentSP, after.CurrentMP, after.CurrentHP)
		}
	case 2:
		if after.CurrentSP != st.MaxSP || after.CurrentMP != st.MaxMP || after.CurrentHP != before.CurrentHP {
			t.Fatalf("level 2: SP %d MP %d HP %d", after.CurrentSP, after.CurrentMP, after.CurrentHP)
		}
	case 3:
		if after.CurrentHP != st.MaxHP || after.CurrentSP != 0 || after.CurrentMP != 0 {
			t.Fatalf("level 3: HP %d SP %d MP %d", after.CurrentHP, after.CurrentSP, after.CurrentMP)
		}
	}
	if s.skillReady(106) {
		t.Fatalf("level %d did not start its reuse", level)
	}
}
