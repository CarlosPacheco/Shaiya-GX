package sdata

import "testing"

func TestGMExperienceStonePresentation(t *testing.T) {
	key := SkillKey{250, 2}
	defs := map[SkillKey]Skill{key: {Name: "Mini Sun EXP StoneED", Level: 2, Icon: 10}}
	item := &Item{Range: 250, AttackTime: 2}
	items := map[ItemKey]*Item{{44, 242}: item}

	NormalizePercentageSkills(defs, false)
	NormalizePercentageItems(items)

	d := defs[key]
	if d.Name != item.Name || d.Description != item.Description || d.Name != "GM EXP Stone" ||
		d.Icon != 10 || d.Level != 2 || d.Effect != 4 || d.Detail != 82 || d.KeepSeconds() != 86400 ||
		d.AbilityType[0] != 35 || d.AbilityValue[0] != 50 || item.Range != 250 || item.AttackTime != 2 {
		t.Fatalf("skill and item presentation disagree: %+v %+v", d, item)
	}
	NormalizePercentageSkills(defs, true)
	NormalizePercentageSkills(defs, false)
	if defs[key] != d {
		t.Fatal("reloading the normalized skill changed it")
	}
}
