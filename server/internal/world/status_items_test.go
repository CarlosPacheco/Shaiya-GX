package world

import (
	"testing"
	"time"

	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/server/internal/db"
)

func TestStatusItemsClearAndProtectNativeStates(t *testing.T) {
	s, add := forgeRuleSession(t)
	defs := db.MustDefs()
	defs.Skills = make(map[db.SkillKey]*db.SkillDef)
	for _, state := range []uint8{2, 7, 8, 9, 14} {
		defs.Skills[db.SkillKey{SkillID: uint16(state), Level: 1}] = &db.SkillDef{StateType: state}
		s.buffs = append(s.buffs, ActiveBuff{ID: uint32(state), SkillID: uint16(state), SkillLevel: 1,
			Effect: typeEffectDebuff, EndsAt: time.Now().Add(time.Minute)})
	}
	s.buffs[4].MapCurse = true
	for _, item := range []db.ItemDef{
		{Type: 25, TypeID: 69, Special: 19, ReqIg: 3},
		{Type: 25, TypeID: 81, Special: 8},
		{Type: 25, TypeID: 234, Special: 26},
	} {
		defs.Items[db.ItemKey{Type: item.Type, TypeID: item.TypeID}] = &item
		add(db.Item{Type: item.Type, TypeID: item.TypeID, Bag: 1, Slot: item.TypeID % 24, Count: 2})
	}
	use := func(id uint8) { s.handleUseItem(proto.UseItemPacket(1, id%24)) }
	use(69)
	if len(s.buffs) != 4 || s.buffs[0].SkillID != 7 || s.items[0].Count != 1 {
		t.Fatalf("Dispel did not remove only its configured states: %+v", s.buffs)
	}
	use(69)
	if s.items[0].Count != 1 {
		t.Fatal("Dispel bypassed shared cooldown")
	}
	use(81)
	if s.blocksStatusSkill(&db.SkillDef{StateType: 7}) || !s.blocksStatusSkill(&db.SkillDef{StateType: 8}) ||
		s.blocksStatusSkill(&db.SkillDef{StateType: 8}) {
		t.Fatal("Death protection must block a single matching skill")
	}
	use(234)
	if len(s.buffs) != 1 || !s.buffs[0].MapCurse || s.statusItemProtection != protectFaint|protectDeath {
		t.Fatal("Perfect Cure did not clear debuffs and arm both protections")
	}
	rows, err := db.AppliedBuffs(s.char.ID)
	if err != nil || len(rows) != 0 {
		t.Fatalf("removed effects still persisted: %v, %v", rows, err)
	}
	t.Cleanup(func() { db.SaveAppliedBuffs(s.char.ID, nil) })
}

func TestStatusItemRuleNativeGroups(t *testing.T) {
	for special, want := range map[uint8]uint16{
		18: 1<<9 | 1<<10 | 1<<11, 19: 1<<1 | 1<<2 | 1<<5 | 1<<6,
		20: 1<<3 | 1<<4, 21: 1<<14 | 1<<15, 22: 1<<12 | 1<<13,
	} {
		states, protection, all := statusItemRule(special)
		if states != want || protection != 0 || all {
			t.Fatalf("unexpected rule for Special %d", special)
		}
	}
}
