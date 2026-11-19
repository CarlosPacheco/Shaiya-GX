package app

import (
	"github.com/Spelunkern/Shaiya-GX/internal/data/sdata"
	"github.com/Spelunkern/Shaiya-GX/internal/data/text"
	"github.com/Spelunkern/Shaiya-GX/internal/game"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"testing"
)

func TestConfirmedPotionReuseGroupsAndOriginalRestoration(t *testing.T) {
	a := &App{clock: 100, state: game.New(), items: map[sdata.ItemKey]*sdata.Item{
		{Type: 25, TypeID: 1}: {Name: "Potion", ReqIg: 1, RecoveryHP: 200},
		{Type: 25, TypeID: 2}: {Name: "Other potion", ReqIg: 1},
		{Type: 25, TypeID: 3}: {Name: "Independent potion", ReqIg: 11},
	}, msg: text.SysMsg{100: "<v><hp> is restored with <i>"}}
	a.state.SelfID = 42
	a.state.Pools = proto.Pools{HP: 500, MP: 50, SP: 70}
	a.state.LastItemUse.ID = 42
	a.confirmedItems(proto.UseItem)
	if len(a.inventory.cooldowns) != 0 {
		t.Fatal("refusal started reuse")
	}
	a.state.LastItemUse.Type, a.state.LastItemUse.TypeID = 25, 1
	a.confirmedItems(proto.UseItem)
	if a.itemCooldown(proto.InventoryItem{Type: 25, TypeID: 2}) != 15 || a.itemCooldown(proto.InventoryItem{Type: 25, TypeID: 3}) != 0 {
		t.Fatal("reuse not shared by ReqIg")
	}
	if len(a.state.Chat) != 0 {
		t.Fatal("restoration predicted before resource snapshot")
	}
	a.state.Pools = proto.Pools{HP: 620, MP: 55, SP: 70}
	a.confirmedItems(proto.CharacterCurrentHitpoints)
	if len(a.state.Chat) != 1 || a.state.Chat[0].MessageID != 100 || a.state.Chat[0].Style != 4 || a.state.Chat[0].Text != "120HP is restored with Potion" {
		t.Fatal("wrong restoration", a.state.Chat)
	}
	a.state.Pools.HP = 700
	a.confirmedItems(proto.CharacterCurrentHitpoints)
	if len(a.state.Chat) != 1 {
		t.Fatal("unrelated regeneration claimed as potion")
	}
	a.clock = 115
	if a.itemCooldown(proto.InventoryItem{Type: 25, TypeID: 1}) != 0 {
		t.Fatal("cooldown boundary")
	}
}

func TestItemTooltipDoesNotInventConsumableStatsOrStackMetadata(t *testing.T) {
	d := &sdata.Item{Name: "Potion", Type: 25, RecoveryHP: 200, Count: 99, Classes: [6]uint8{1}, RequiredStats: [6]uint16{12}, Description: "Restores 200 HP."}
	rows := itemDetailLines(d, nil, 12, "", nil)
	if len(rows) != 2 || rows[0].text != "Potion" || rows[1].text != d.Description {
		t.Fatal("consumable has invented equipment/stack metadata", rows)
	}
	if rows[0].color != ([4]float32{1, 1, 0, 1}) {
		t.Fatal("native title colour")
	}
}

func TestBuffItemOnlyReservesActionAfterConfirmation(t *testing.T) {
	a := &App{clock: 100, state: game.New(), items: map[sdata.ItemKey]*sdata.Item{{Type: 25, TypeID: 175}: {Range: 211, AttackTime: 1}}, skillDefs: map[sdata.SkillKey]sdata.Skill{{ID: 211, Level: 1}: {Effect: 4}}}
	a.state.SelfID = 42
	it := proto.InventoryItem{Type: 25, TypeID: 175}
	a.itemCooldown(it)
	if a.skills.actionKnown {
		t.Fatal("rendering an item icon reserved combat")
	}
	a.state.LastItemUse.ID = 42
	a.confirmedItems(proto.UseItem)
	if a.skills.actionKnown {
		t.Fatal("rejected item reserved combat")
	}
	a.state.LastItemUse.Type, a.state.LastItemUse.TypeID = 25, 175
	a.confirmedItems(proto.UseItem)
	if !a.skills.actionKnown || a.skills.lastActionAt != 100 {
		t.Fatal("confirmed item did not reserve combat")
	}
}

func TestItemCooldownResetClearsConsumableAndMountTimers(t *testing.T) {
	a := &App{clock: 100, state: game.New(), items: map[sdata.ItemKey]*sdata.Item{
		{Type: 25, TypeID: 1}: {ReqIg: 1},
	}}
	a.inventory.cooldowns = map[uint8]float64{1: 115, 11: 102}
	a.vehicle.cooldownEnd = 160
	ev, err := a.state.Apply(proto.NewWriter(proto.GXItemCooldownReset).Bytes())
	if err != nil {
		t.Fatal(err)
	}
	a.confirmedItems(ev.Op)
	if len(a.inventory.cooldowns) != 0 || a.itemCooldown(proto.InventoryItem{Type: 42}) != 0 {
		t.Fatal("item or mount cooldown survived reset")
	}
}
