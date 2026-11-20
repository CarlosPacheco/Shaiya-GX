package world

import (
	"testing"

	"github.com/Spelunkern/Shaiya-GX/server/internal/db"
)

func TestReflectorCategories(t *testing.T) {
	old := db.MustDefs()
	t.Cleanup(func() { db.SetDefsForTest(old) })
	mk := func(id uint16, v [3]int16) *db.SkillDef {
		return &db.SkillDef{SkillID: id, SkillLevel: 1, TypeEffect: typeEffectBuff, TypeDetail: 83, SuccessType: 1, SuccessValue: 100, AbilityValue: v}
	}
	defs := &db.Defs{Skills: map[db.SkillKey]*db.SkillDef{
		{SkillID: 89, Level: 1}: mk(89, [3]int16{1, 0, 0}),
		{SkillID: 86, Level: 1}: mk(86, [3]int16{0, 0, 1}),
	}}
	db.SetDefsForTest(defs)
	for _, c := range []struct {
		skill uint16
		kind  int
		want  bool
	}{{89, attackMelee, true}, {89, attackMagic, false}, {86, attackMagic, true}, {86, attackMelee, false}} {
		s := &Session{}
		s.buffs = []ActiveBuff{{SkillID: c.skill, SkillLevel: 1, Effect: typeEffectBuff, Detail: 83}}
		hit := false
		for i := 0; i < 50; i++ {
			hit = hit || s.reflects(c.kind, nil)
		}
		if hit != c.want {
			t.Fatalf("skill %d kind %d reflected=%v want %v", c.skill, c.kind, hit, c.want)
		}
	}
}

func TestWarehouseCallCannotBeCancelled(t *testing.T) {
	old := db.MustDefs()
	t.Cleanup(func() { db.SetDefsForTest(old) })
	db.SetDefsForTest(&db.Defs{Skills: map[db.SkillKey]*db.SkillDef{
		{SkillID: 228, Level: 1}: {SkillID: 228, SkillLevel: 1, TypeEffect: typeEffectBuff, TypeDetail: 82, AbilityType: [3]uint8{39, 0, 0}},
	}})
	b := ActiveBuff{SkillID: 228, SkillLevel: 1, Effect: typeEffectBuff, Detail: 82}
	if !isRemoteWarehouse(&b) {
		t.Fatal("Warehouse Call must be protected from manual cancel")
	}
}
