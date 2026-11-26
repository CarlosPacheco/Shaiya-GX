package app

import (
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/data/sdata"

	"github.com/Spelunkern/Shaiya-GX/internal/data/text"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/ui"
)

func TestWarehouseCallClickReopensPersonalBankAndStillAllowsDragging(t *testing.T) {
	a := inputApp()
	var err error
	a.bankUI, err = ui.NewWarehouse(inputTextures{})
	if err != nil {
		t.Fatal(err)
	}
	a.state.MapID = 51
	active := proto.ActiveBuff{Skill: warehouseCallSkill, Instance: 3}
	a.state.Buffs = []proto.ActiveBuff{active}
	a.buffs.rows = []buffBadge{{buff: active}}
	a.bank = warehouseView{npcID: 1, guild: true, page: 4}
	a.state.Warehouse = map[uint8]proto.InventoryItem{0: {Type: 25, TypeID: 4}}
	a.state.Guild.Bank = map[uint8]proto.InventoryItem{0: {Type: 5, TypeID: 1}}
	const w, h = 1024, 768

	clickBuff := func(dx int) {
		x, y := a.buffSlot(w, h, 0)
		a.worldControls(&platform.Input{MouseX: x + 5, MouseY: y + 5, Pressed: true, MouseDown: true}, w, h)
		a.worldControls(&platform.Input{MouseX: x + 5 + dx, MouseY: y + 5, Clicked: true}, w, h)
	}
	clickBuff(0)
	if !a.remoteWarehouseOpen() || !a.inventory.open || a.bank.page != 0 {
		t.Fatal("clicking Warehouse Call did not open the personal warehouse and inventory")
	}
	a.updateWarehouse(&platform.Input{}, w, h)
	if a.warehouseBag() != proto.WarehouseBag || a.warehouseItems()[0] != a.state.Warehouse[0] {
		t.Fatal("remote warehouse used guild storage inside a guild house")
	}
	a.closeWarehouse()
	clickBuff(-80)
	if a.remoteWarehouseOpen() || !a.buffLayout.Position[0].Moved {
		t.Fatal("dragging the Warehouse Call buff opened the warehouse or failed to move the strip")
	}
	clickBuff(0)
	if !a.remoteWarehouseOpen() {
		t.Fatal("the moved buff did not reopen the warehouse")
	}
	a.state.Buffs = nil
	if a.remoteWarehouseOpen() {
		t.Fatal("the window remained open after the effect expired")
	}
}

func TestWarehouseCallTooltipsExplainRemoteAccess(t *testing.T) {
	a := inputApp()
	key := sdata.SkillKey{ID: warehouseCallSkill, Level: 1}
	def := sdata.Skill{Name: "Warehouse Call", Description: "Legacy ? ? ?", Effect: 4, KeepTime: 300}
	a.skillDefs = map[sdata.SkillKey]sdata.Skill{key: def}
	a.queueSkillTooltip(key)
	if a.tooltipDescription != warehouseCallDescription || len(a.tooltip) != 2 || a.tooltip[1].text != "Duration: 5m" {
		t.Fatalf("wrong skill tooltip: %q %+v", a.tooltipDescription, a.tooltip)
	}
	a.buffs.rows = []buffBadge{{buff: proto.ActiveBuff{Skill: warehouseCallSkill, Level: 1}, def: def, until: 20, timed: true}}
	a.lastMouseX, a.lastMouseY = a.buffSlot(1024, 768, 0)
	a.drawBuffTooltip(1024, 768)
	if a.tooltipDescription != warehouseCallDescription+"\n00:00:20" || len(a.tooltip) != 1 {
		t.Fatalf("wrong active buff tooltip: %q %+v", a.tooltipDescription, a.tooltip)
	}
}

func TestWarehouseInsufficientGoldUsesOriginalMessage(t *testing.T) {
	a := inputApp()
	a.bankUI, _ = ui.NewWarehouse(inputTextures{})
	a.state.Buffs = []proto.ActiveBuff{{Skill: warehouseCallSkill}}
	a.remoteBank, a.state.WarehouseReady = true, true
	a.state.Warehouse = map[uint8]proto.InventoryItem{40: {Bag: proto.WarehouseBag, Slot: 40, Type: 25, TypeID: 4, Count: 2}}
	a.items = map[sdata.ItemKey]*sdata.Item{{Type: 25, TypeID: 4}: {SellPrice: 200}}
	a.msg = text.SysMsg{proto.MsgInsufficientGold: "Insufficient gold"}
	a.state.Details.Money = 19
	a.bank.dragging, a.bank.selected = true, true
	a.moveWarehouseItem([2]uint8{proto.WarehouseBag, 40}, [2]uint8{1, 0})
	if len(a.state.Chat) != 1 || a.state.Chat[0].Text != "Insufficient gold" ||
		a.state.Chat[0].MessageID != proto.MsgInsufficientGold || a.state.Chat[0].Style != 17 ||
		a.inventory.pending || a.bank.dragging || a.state.Warehouse[40].Count != 2 || a.state.Details.Money != 19 {
		t.Fatalf("unexpected warehouse refusal: %+v", a.state.Chat)
	}
}
