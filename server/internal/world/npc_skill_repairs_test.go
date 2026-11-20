package world

import (
	"strconv"
	"testing"
	"time"

	"github.com/Spelunkern/Shaiya-GX/server/internal/db"
)

func TestRepairedNPCSkillsUseStandardDebuffs(t *testing.T) {
	for _, tc := range []struct {
		id, detail uint16
		seconds    uint32
		hpPercent  uint32
	}{
		{100, detailDegeneration, 15, 0},
		{172, detailStun, 15, 25},
		{344, detailDeathSentence, 20, 70},
	} {
		t.Run(strconv.Itoa(int(tc.id)), func(t *testing.T) {
			base, defs := lapisSession(t)
			id, mob := lapisMob(t, base, defs)
			mob.Dex, mob.Wis = 10000, 10000
			base.char.Level = mob.Level
			base.char.CurrentHP = base.stats().MaxHP
			s, _ := gmWireSession(t, *base.char, false)
			s.items = base.items
			s.player = RegisterPlayer(*s.char, nil, false, s.faction, s.out, s.logger)
			t.Cleanup(s.player.Unregister)
			skill := defs.NPCSkill(tc.id)
			if skill == nil || skill.TypeEffect != typeEffectDebuff || skill.TypeDetail != tc.detail || skill.KeepSeconds != tc.seconds {
				t.Fatalf("unexpected NPC skill %d: %+v", tc.id, skill)
			}
			// Preserve the configured chance in the catalog and force success here
			// so the application and expiry assertions are deterministic.
			fixedRoll := *skill
			fixedRoll.SuccessType, fixedRoll.SuccessValue = 1, 100
			defs.NPCSkills[db.SkillKey{SkillID: tc.id, Level: 1}] = &fixedRoll
			before := s.stats()
			s.applyIncomingMobDamage(pendingDamage{MobID: mob.MobID, MobGlobalID: id, SkillID: tc.id})
			if got := before.CurrentHP - s.stats().CurrentHP; got != before.MaxHP*tc.hpPercent/100 {
				t.Fatalf("NPC skill %d initial loss = %d", tc.id, got)
			}
			if len(s.buffs) != 1 || s.buffs[0].SkillID != tc.id {
				t.Fatalf("NPC skill %d failed to apply: %+v", tc.id, s.buffs)
			}
			switch tc.id {
			case 100:
				if s.transformedMob() != 60 || !s.cannotAct() {
					t.Fatal("Porcine Party did not apply the pig transformation")
				}
			case 172:
				if !s.actionBlocked() || s.speeds().move != 255 {
					t.Fatal("Ice Stun did not stop movement and actions")
				}
			case 344:
				expires := s.buffs[0].EndsAt
				s.processBuffsAt(expires.Add(-time.Second))
				if s.dead || s.stats().CurrentHP == 0 {
					t.Fatal("Wide Death Sentence killed before expiration")
				}
				s.processBuffsAt(expires)
				if !s.dead || s.stats().CurrentHP != 0 {
					t.Fatal("Wide Death Sentence failed to kill on expiration")
				}
			}
		})
	}
}
