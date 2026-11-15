package app

import (
	"github.com/Spelunkern/Shaiya-GX/internal/data/sdata"
	"github.com/Spelunkern/Shaiya-GX/internal/game"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"testing"
)

func TestServerChestFitCorrectsStaleCatalog(t *testing.T) {
	a := &App{state: game.New(), items: map[sdata.ItemKey]*sdata.Item{{Type: 17, TypeID: 216}: {Type: 17, TypeID: 216, Model: 20, ReqVG: 0}, {Type: 24, TypeID: 244}: {Type: 24, TypeID: 244, Model: 21}}}
	c := proto.CharacterSlot{ID: 42, Class: 5}
	c.EquipType[1], c.EquipTypeID[1] = 17, 216
	c.EquipType[7], c.EquipTypeID[7] = 24, 244
	if a.appearance(&c).CloakFit != 0 {
		t.Fatal("bad fallback")
	}
	if _, err := a.state.Apply(proto.ChestCloakFitsPacket([]proto.ChestCloakFit{{Type: 17, TypeID: 216, Fit: 7}})); err != nil {
		t.Fatal(err)
	}
	if a.appearance(&c).CloakFit != 7 {
		t.Fatal("stale SData hid the shoulder under armor")
	}
	if a.items[sdata.ItemKey{Type: 17, TypeID: 216}].ReqVG != 0 {
		t.Fatal("cosmetic correction mutated general item rules")
	}
	if _, err := a.state.Apply(proto.ChestCloakFitsPacket([]proto.ChestCloakFit{{Type: 17, TypeID: 216, Fit: 0}, {Type: 17, TypeID: 36, Fit: 8}})); err == nil || a.appearance(&c).CloakFit != 7 {
		t.Fatal("malformed catalog partially applied")
	}
	if _, err := a.state.Apply(proto.ChestCloakFitsPacket([]proto.ChestCloakFit{{Type: 17, TypeID: 216, Fit: 0}})); err != nil || a.appearance(&c).CloakFit != 0 {
		t.Fatal("authoritative zero was ignored")
	}
}
