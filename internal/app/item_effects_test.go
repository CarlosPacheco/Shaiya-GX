package app

import (
	"github.com/Spelunkern/Shaiya-GX/internal/data/sdata"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"testing"
)

func TestItemEffectsNPCNamespaceAndIntegratedLapis(t *testing.T) {
	a := inputApp()
	a.npcSkillDefs = map[sdata.SkillKey]sdata.Skill{{ID: 205, Level: 1}: {Name: "NPC", Icon: 23, Effect: 3}}
	a.skillDefs = map[sdata.SkillKey]sdata.Skill{{ID: 205, Level: 1}: {Name: "Player", Icon: 99, Effect: 2}}
	gem := &sdata.Item{Type: 30, TypeID: 151, Special: 205}
	a.items = map[sdata.ItemKey]*sdata.Item{{Type: 30, TypeID: 151}: gem}
	for _, tc := range []struct {
		d  *sdata.Item
		it *proto.InventoryItem
	}{
		{gem, nil}, {&sdata.Item{Type: 12}, &proto.InventoryItem{Gems: [6]uint8{151, 151}}},
	} {
		got, _ := a.itemEffects(tc.d, tc.it)
		if len(got) != 1 || got[0].Icon != 23 {
			t.Fatalf("effects: %+v", got)
		}
	}
	delete(a.npcSkillDefs, sdata.SkillKey{ID: 205, Level: 1})
	if got, _ := a.itemEffects(gem, nil); len(got) != 0 {
		t.Fatalf("player fallback: %+v", got)
	}
}

func TestAntiLapisUsesSeparateTooltipSkillRow(t *testing.T) {
	a := inputApp()
	a.npcSkillDefs = map[sdata.SkillKey]sdata.Skill{
		{ID: 1, Level: 1}:   {Name: "Sleep", Icon: 23, Effect: 3, SuccessValue: 8},
		{ID: 205, Level: 1}: {Name: "Pain", Icon: 24, Effect: 3},
	}
	anti := &sdata.Item{Type: 30, TypeID: 111, Special: 1}
	proc := &sdata.Item{Type: 30, TypeID: 151, Special: 205}
	a.items = map[sdata.ItemKey]*sdata.Item{{Type: 30, TypeID: 111}: anti, {Type: 30, TypeID: 151}: proc}
	effects, immune := a.itemEffects(anti, nil)
	if len(effects) != 0 || len(immune) != 1 || immune[0].Icon != 23 {
		t.Fatalf("standalone anti-lapis effects %v immunities %v", effects, immune)
	}
	effects, immune = a.itemEffects(&sdata.Item{Type: 34}, &proto.InventoryItem{Gems: [6]uint8{111, 151, 111}})
	if len(effects) != 1 || effects[0].Icon != 24 || len(immune) != 1 || immune[0].Icon != 23 {
		t.Fatalf("linked lapis effects %v immunities %v", effects, immune)
	}
}
