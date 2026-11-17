package world

import (
	"github.com/Spelunkern/Shaiya-GX/server/internal/db"
	"testing"
	"time"
)

func TestWeaponEnchantDirectHealing(t *testing.T) {
	s, _ := forgeRuleSession(t)
	def := db.MustDefs().Item(1, 1)
	def.ConstHP = 2000
	s.char.Wis = 13
	s.skillCooldowns = map[uint16]time.Time{}
	s.player = RegisterPlayer(*s.char, nil, false, s.faction, s.out, s.logger)
	t.Cleanup(s.player.Unregister)
	skill := &db.SkillDef{SkillID: 64001, SkillLevel: 1, TypeEffect: typeEffectHeal, TypeDetail: detailHeal, TargetType: targetTypeSelf, HealHP: 40}
	k := db.CharacterSkill{SkillID: skill.SkillID, Level: 1}
	for _, tt := range []struct {
		craft string
		bag   uint8
		bonus uint32
	}{
		{"00000000000000000000", 0, 0},
		{"00000000000000000001", 0, 7},
		{"00000000000000000020", 0, 311},
		{"00000000000000000020", 1, 0},
		{"00000000000000000070", 0, 0},
	} {
		s.items = []db.Item{{Bag: tt.bag, Slot: weaponSlot, Type: 1, TypeID: 1, CraftName: tt.craft}}
		s.char.CurrentHP = 1
		st := s.stats()
		if st.HealingBonus != tt.bonus || s.equipment().ItemAttack != tt.bonus {
			t.Fatalf("craft=%s bag=%d bonus=%d want %d", tt.craft, tt.bag, st.HealingBonus, tt.bonus)
		}
		s.friendlySkill(k, skill, s.char.ID, st)
		if got, want := s.char.CurrentHP, uint32(1+40+26)+tt.bonus; got != want {
			t.Fatalf("direct heal=%d want %d", got, want)
		}
	}
	// No new HP healing for resource-only spells or a cure.
	s.items[0].CraftName = "00000000000000000020"
	for _, detail := range []uint16{detailHeal, detailCure} {
		s.char.CurrentHP = 1
		skill.HealHP, skill.TypeDetail = 0, detail
		s.friendlySkill(k, skill, s.char.ID, s.stats())
		if s.char.CurrentHP != 1 {
			t.Fatal("resource-only/cure spell gained HP healing")
		}
	}
	// Periodic healing and generic restoration must not use the enchant bonus.
	periodic := &db.SkillDef{SkillID: 64002, SkillLevel: 1, TypeEffect: typeEffectBuff, TimeHealHP: 11}
	b := newBuff(1, periodic, time.Now().Add(time.Minute), s.log)
	b.nextTickAt = time.Now().Add(-time.Second)
	s.buffs = []ActiveBuff{b}
	s.processBuffs()
	if s.char.CurrentHP != 12 {
		t.Fatalf("periodic heal scaled: HP=%d", s.char.CurrentHP)
	}
	hp, _, _ := s.healDelta(9, 0, 0)
	if hp != 9 {
		t.Fatalf("generic restoration scaled: %d", hp)
	}
}

func TestWeaponHealingRequiresEligibleWeapon(t *testing.T) {
	old := db.MustDefs()
	t.Cleanup(func() { db.SetDefsForTest(old) })
	for _, tt := range []struct {
		typ, server uint8
		req         int16
	}{
		{17, 1, 1}, {22, 1, 1}, {1, 0, 1}, {1, 1, 0}, {1, 1, 100},
	} {
		db.SetDefsForTest(&db.Defs{Items: map[db.ItemKey]*db.ItemDef{{Type: tt.typ, TypeID: 1}: {Type: tt.typ, TypeID: 1, Server: tt.server, ReqWis: int16(tt.req)}}})
		e := GetEquipmentStats([]db.Item{{Bag: 0, Type: tt.typ, TypeID: 1, CraftName: "00000000000000000020"}})
		if e.HealingBonus != 0 {
			t.Fatalf("nonweapon/ineligible item grants healing: %+v", tt)
		}
	}
}
