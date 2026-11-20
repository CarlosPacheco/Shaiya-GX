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
		{89, 1}: mk(89, [3]int16{1, 0, 0}),
		{86, 1}: mk(86, [3]int16{0, 0, 1}),
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
